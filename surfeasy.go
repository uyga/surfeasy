package main

import (
    "net/http"
    "fmt"
    "bytes"
    "crypto/rand"
    "crypto/sha1"
    "io/ioutil"
    "encoding/json"
    "time"
)

var (
    cookies []*http.Cookie
    device_hash string
    device_id string
    device_id_hash string
    device_password string
)

const (
    CLIENT_TYPE = "se0310"
    API_KEY = "AE4CA57D1E3C0E6711C53416BFA0988F08D41B428D26D053A4C46EC72A79B9E7"
    OS = "Windows"
)

func post(url string, data []byte) string {
    server := fmt.Sprintf("https://api.surfeasy.com/%s", url)
    req, err := http.NewRequest("POST", server, bytes.NewBuffer(data))
    req.Header.Set("SE-Client-Type", CLIENT_TYPE)
    req.Header.Set("SE-Client-API-Key", API_KEY)
    req.Header.Set("SE-Operating-System", OS)
    req.Header.Set("Content-Type", "application/json")
    if len(cookies) > 0 {
        for i:=0; i<len(cookies); i++ {
            req.AddCookie(cookies[i]);
        }
    }

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    cookies = resp.Cookies()
    body, _ := ioutil.ReadAll(resp.Body)
    return string(body)
}

func uuid() string {

    b := make([]byte, 16)
    _, err := rand.Read(b)
    if err != nil {
        panic(err)
    }

    uuid := fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

    return uuid
}

func getData(body string) map[string]interface{} {
    var dat map[string]interface{}
    if err := json.Unmarshal([]byte(body), &dat); err != nil {
        panic(err)
    }
    dat = dat["data"].(map[string]interface{})
    return dat
} 

func registerSubscriber() string {
    url := "/v2/register_subscriber"
    email := fmt.Sprintf("%s@%s.surfeasy.vpn", uuid(), CLIENT_TYPE)
    password := uuid()
    pass_hash := fmt.Sprintf("%x", sha1.Sum([]byte(password)))
    var data = []byte(`{"email":"` + email + `", "password":"` + pass_hash + `"}`)
    body := post(url, data)
    return body
}

func registerDevice() string {
    url := "/v2/register_device"
    var data = []byte(`{"client_type":"` + CLIENT_TYPE + `", "device_hash":"` + device_hash + `", "device_name":"Opera-Browser-Client"}`)
    body := post(url, data)
    parsed := getData(body)
    device_id = parsed["device_id"].(string)
    device_id_hash = fmt.Sprintf("%x", sha1.Sum([]byte(device_id)))
    device_password = parsed["device_password"].(string)
    return body
}

func geoList() (map[int]string) {
    url := "/v2/geo_list"
    var data = []byte(`{"device_id":"` + device_id_hash + `"}`)
    body := post(url, data)
    parsed := getData(body)
    geos := parsed["geos"].([]interface{})
    var countries = map[int]string{}
    for key, country := range geos {
        countries[key] = country.(map[string]interface{})["country_code"].(string)
    }
    return countries
}

func discover(country_code string) map[string]string {
    auth := fmt.Sprintf("%s:%s", device_id_hash, device_password)
    url := "/v2/discover"
    var data = []byte(`{"serial_no":"` + device_id_hash + `", "requested_geo":"` + country_code + `"}`)
    body := post(url, data)
    parsed := getData(body)
    var proxies = map[string]string{}
    for key, proxy := range parsed["ips"].([]interface{}) {
        ports := proxy.(map[string]interface{})["ports"]
        ip := proxy.(map[string]interface{})["ip"]
        for p, port := range ports.([]interface{}) {
            proxies[fmt.Sprintf("%v%v", key, p)] = fmt.Sprintf("%s:%v", ip, port)
            fmt.Printf("curl -kx https://%s@%s:%v --proxy-insecure %%s\n", auth, ip, port)
        }
    }
    return proxies
}

func main() {
    device_hash = fmt.Sprintf("%x", sha1.Sum([]byte(time.Now().Format(time.RFC850))))
    registerSubscriber()
    registerDevice()
    countries := geoList()
    for _, country_code := range countries {
        fmt.Println("[COUNTRY]: ", country_code)
        discover(country_code)
    }
}





