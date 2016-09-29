package main

import (
    //"log"
    //"time"
    "os" 
    //"io"
    //"strings"
    "fmt"
    "net/http"
    "encoding/json"
    "strconv"
    "sort"
)

var COOKIE = os.Getenv("NBBCOOKIE")
var DOMAIN = ""

type User struct{
    userslug string
    posts int
}
type ByPosts []User

func (a ByPosts) Len() int           { return len(a) }
func (a ByPosts) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByPosts) Less(i, j int) bool { return a[i].posts > a[j].posts }

// Sum number of post of each user on a single page
func sumTopicPageSeq(tid string, p string) (map[string]int){
    result := make(map[string]int)
    client := &http.Client{}

    req, err := http.NewRequest("GET", "http://"+ DOMAIN +"/api/topic/"+ tid +"/?page="+ p, nil)
    if err != nil{
        fmt.Printf("Error preparing request for "+ string(p) + " for topic "+ tid + "\n")
        return result
    }
    req.Header.Add("Cookie", COOKIE)
    resp, err := client.Do(req)
    if err != nil{
        fmt.Printf("Error obtaining page "+ string(p) + " for topic "+ tid + "\n")
        return result
    }

    defer resp.Body.Close()
    //io.Copy(os.Stdout, resp.Body)

    var topicPage map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&topicPage)

    posts := topicPage["posts"].([]interface{})
    for i := range posts{
        post := posts[i].(map[string]interface{})
        user := post["user"].(map[string]interface{})
        userslug := user["userslug"].(string)
        //fmt.Printf("%+v\n", userslug)

        result[userslug] = result[userslug] + 1
    }

    return result

}

// Sum number of post of each user on a single page in parallel
func sumTopicPage(tid string, p string, c chan map[string]int){
    result := make(map[string]int)

    client := &http.Client{}
    req, err := http.NewRequest("GET", "https://"+ DOMAIN +"/api/topic/"+ tid +"/?page="+ p, nil)
    if err != nil{
        fmt.Printf("Error preparing request for "+ string(p) + " for topic "+ tid + "\n")
        c <- result
        return
    }
    req.Header.Add("Cookie", COOKIE)
    resp, err := client.Do(req)
    if err != nil{
        fmt.Printf("Error obtaining page "+ string(p) + " for topic "+ tid + ". Retrying..\n")
        sumTopicPage(tid, p, c)
        return
    }

    defer resp.Body.Close()
    //io.Copy(os.Stdout, resp.Body)

    var topicPage map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&topicPage)

    posts := topicPage["posts"].([]interface{})
    for i := range posts{
        post := posts[i].(map[string]interface{})
        user := post["user"].(map[string]interface{})
        userslug := user["userslug"].(string)

        result[userslug] = result[userslug] + 1
    }

    c <- result
    return

}

func getTopicPages(tid string) int{
    result := 0
    client := &http.Client{}

    req, err := http.NewRequest("GET", "https://"+ DOMAIN +"/api/topic/"+ tid +"/?page=1", nil)
    if err != nil{
        fmt.Printf("Error obtaining pages num for topic "+ tid)
        return result
    }
    req.Header.Add("Cookie", COOKIE)
    resp, err := client.Do(req)
    if err != nil{
        fmt.Printf("Error obtaining pages num for topic "+ tid)
        return result
    }

    defer resp.Body.Close()
    //io.Copy(os.Stdout, resp.Body)

    var topicPage map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&topicPage)

    pagination := topicPage["pagination"].(map[string]interface{})
    pages := int(pagination["pageCount"].(float64))

    return pages

}


func processTopic(topicId string) ([]User){

    pages := getTopicPages(topicId)
    fmt.Printf("%d pages to process..\n", pages)
    sumPages := make([]map[string]int, pages)

    c := make(chan map[string]int)
    for i := range sumPages{
        iToString := strconv.Itoa(i)
        //sumPages[i] = sumTopicPageSeq(topicId, iToString) // sequential
        go sumTopicPage(topicId, iToString, c)
    }

    mapResult := make(map[string]int)
    for i := range sumPages{
        sumPages[i] = <-c
        for j := range sumPages[i]{
            mapResult[j] = mapResult[j] + sumPages[i][j]
        }
    }

    result := make([]User, 0)
    for i := range mapResult{
        result = append(result, User{ i, mapResult[i] })
    }

    sort.Sort(ByPosts(result))
    return result
}


func main() {

    if len(os.Args) < 3{
        fmt.Printf("Usage: %s <forum_domain> <topic_id>\n", os.Args[0])
        fmt.Printf("Important: Set NBBCOOKIE environment variable if needed.\n")
        return
    }

    DOMAIN = os.Args[1]
    tid := os.Args[2]
    fmt.Printf("Processing %s topic: %s..\n", DOMAIN, tid)

    r := processTopic(tid)
    //r := processTopic("183") // first flood topic in exodo
    //fmt.Printf("%+v\n", r)

    for i := range r{
        fmt.Printf("%d. "+ r[i].userslug +" - %d posts\n", i+1, r[i].posts)
    }
}


