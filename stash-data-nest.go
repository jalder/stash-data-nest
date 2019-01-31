package main

import (
    "os"
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    bson "github.com/mongodb/mongo-go-driver/bson"
    mongo "github.com/mongodb/mongo-go-driver/mongo"
    context "golang.org/x/net/context"
    "regexp"
)

func main() {
    url := "https://developer-api.nest.com/"
    req, _ := http.NewRequest(http.MethodGet, url, nil)

    token := os.Getenv("NESTAUTHTOKEN")

    req.Header.Add(
        "Authorization",
        fmt.Sprintf("Bearer %s", token),
    )

    customClient := http.Client {
        CheckRedirect: func(redirRequest *http.Request, via []*http.Request) error {
            // Go's http.DefaultClient does not forward headers when a redirect 3xx
            // response is received. Thus, the header (which in this case contains the
            // Authorization token) needs to be passed forward to the redirect
            // destinations.
            redirRequest.Header = req.Header

            // Go's http.DefaultClient allows 10 redirects before returning an
            // an error. We have mimicked this default behavior.
            if len(via) >= 10 {
                return errors.New("stopped after 10 redirects")
            }
            return nil
        },
    }

    response, _ := customClient.Do(req)

    if response.StatusCode != 200 {
        panic(fmt.Sprintf(
            "Expected a 200 status code; got a %d",
            response.StatusCode,
        ))
    }

    defer response.Body.Close()
    body, _ := ioutil.ReadAll(response.Body)

    mongodbUrl := os.Getenv("MDBATLASCONN")
    client, err := mongo.Connect(context.Background(), mongodbUrl, nil)
    if err != nil {
        fmt.Println("error connecting")
    }
    db := client.Database("nest")
    coll := db.Collection("reads")
    var re = regexp.MustCompile(`"([0-9]{4})-(1[0-2]|0[1-9])-([0-9]{2}T)([0-9]{2}):([0-9]{2}):([0-9]{2}).([0-9]{3}Z)"`)
    s := re.ReplaceAllString(string(body), `{"$$date":"$1-$2-$3$4:$5:$6.$7"}`)
    var b interface{}
    if err := bson.UnmarshalExtJSON([]byte(s), false, &b); err != nil {
        fmt.Println("error json string to bson")
        fmt.Println(err)
    }
    if _, err := coll.InsertOne(context.TODO(),b); err != nil {
        // handle error
        fmt.Println("error inserting")
    }
}

