package main

import (
    "embed"
    "encoding/hex"
    "html"
    "image"
    "image/color"
    "image/draw"
    "image/png"
    "log"
    "math/rand"
    "net/http"
    "strconv"
    "sync"
    "text/template"
    "time"
)

type Msg struct {
    FromName string
    Time     time.Time
    Content  string
}

func (m Msg) TimeStr() string {
    return m.Time.Format(time.RFC850)
}

type User struct {
    Name           string
    PrivateId      string
    Seen           chan struct{}
    PreferredStyle string
}

type Style struct {
    Name     string
    Selected bool
}

type TemplateParams struct {
    Name   string
    Theme  string
    OutBox []Msg
    Styles []Style
}

func genFavicon(primary, secondary, accent color.Color) *image.RGBA {
    m := image.NewRGBA(image.Rect(0, 0, 16, 16))
    draw.Draw(m, m.Bounds(), &image.Uniform{primary}, image.ZP, draw.Src)
    for i := 1; i < 15; i++ {
        for j := 1; j < 15; j++ {
            x := 8 - i
            y := 8 - j
            if x*x+y*y <= 7*7 {
                m.Set(i, j, secondary)
            }
        }
    }
    for i := 1; i < 15; i++ {
        for j := 1; j < 15; j++ {
            x := 5 - i
            y := 5 - j
            if x*x+y*y <= 3*3 {
                m.Set(i, j, accent)
            }
        }
    }
    return m
}

//go:embed views
//go:embed libs/htmx.min.js
var embedFS embed.FS

func makeHandler() func(w http.ResponseWriter, r *http.Request) {
    store := Storage{users: make(map[string]User)}

    tmpl, err := template.ParseFS(embedFS, "views/index.html")
    if err != nil {
        panic(err)
    }

    htmxLib, err := embedFS.ReadFile("libs/htmx.min.js")
    if err != nil {
        panic(err)
    }

    style_brutal, err := template.ParseFS(embedFS, "views/style.css")
    if err != nil {
        panic(err)
    }

    requestId := func(r *http.Request) string {
        c, err := r.Cookie("id")
        if err == nil {
            return c.Value
        }
        return ""
    }

    black := color.RGBA{0, 0, 0, 255}
    green := color.RGBA{0, 255, 0, 255}
    white := color.RGBA{255, 255, 255, 255}
    blue := color.RGBA{0, 0, 255, 255}
    lightblue := color.RGBA{0, 255, 255, 255}

    favicon_brutal := genFavicon(black, green, white)

    favicon_milky := genFavicon(blue, lightblue, white)

    return func(w http.ResponseWriter, r *http.Request) {

        u2 := store.getUser(requestId(r))
        if u2 == nil && (r.URL.Path != "/" || r.Method != "GET") {
            w.Header().Set("HX-Redirect", "/")
            return
        }

        if r.URL.Path == "/" && r.Method == "GET" {
            if u2 == nil {
                u2 = store.createUser()
                http.SetCookie(w, &http.Cookie{Name: "id", Value: u2.PrivateId})
            }
            theme := u2.PreferredStyle
            out := store.getMsgs()
            styles := []Style{
                {Name: "milky", Selected: u2.PreferredStyle == "milky"},
                {Name: "brutal", Selected: u2.PreferredStyle == "brutal"},
            }
            err := tmpl.Execute(w, TemplateParams{Styles: styles, Name: u2.Name, Theme: theme, OutBox: out})
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/style" && r.Method == "PUT" {
            style := html.EscapeString(r.PostFormValue("preferred"))
            store.putStyle(u2.PrivateId, style)
            w.Header().Set("HX-Redirect", "/")
        } else if r.URL.Path == "/messages" && r.Method == "GET" {
            out := store.getMsgs()
            err := tmpl.ExecuteTemplate(w, "messages", out)
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/messages" && r.Method == "POST" {
            store.postMsg(u2.PrivateId, html.EscapeString(r.PostFormValue("message")))
            err := tmpl.ExecuteTemplate(w, "input", struct{}{})
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/name/edit" && r.Method == "GET" {
            err := tmpl.ExecuteTemplate(w, "name/edit", u2.Name)
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/name" && r.Method == "GET" {
            err := tmpl.ExecuteTemplate(w, "name", u2.Name)
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/name" && r.Method == "PUT" {
            newName := html.EscapeString(r.PostFormValue("name"))
            store.putName(u2.PrivateId, newName)
            err := tmpl.ExecuteTemplate(w, "name", newName)
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/htmx.min.js" && r.Method == "GET" {
            w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
            _, err := w.Write(htmxLib)
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/style_milky.css" && r.Method == "GET" {
            w.Header().Set("Content-Type", "text/css")
            err := style_brutal.Execute(w, "milky")
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/style_brutal.css" && r.Method == "GET" {
            w.Header().Set("Content-Type", "text/css")
            err := style_brutal.Execute(w, "brutal")
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/favicon_brutal.ico" && r.Method == "GET" {
            w.Header().Set("Content-Type", "image/x-icon")
            err := png.Encode(w, favicon_brutal)
            if err != nil {
                panic(err)
            }
        } else if r.URL.Path == "/favicon_milky.ico" && r.Method == "GET" {
            w.Header().Set("Content-Type", "image/x-icon")
            err := png.Encode(w, favicon_milky)
            if err != nil {
                panic(err)
            }
        } else {
            log.Printf("Unexpected Request %v", r)
        }

    }
}

func serverMsg(s string) Msg {
    return Msg{
        FromName: "Server",
        Time:     time.Now(),
        Content:  s,
    }
}

type Storage struct {
    lock  sync.RWMutex
    users map[string]User
    box   []Msg
}

func (s *Storage) getMsgs() []Msg {
    s.lock.RLock()
    cp := make([]Msg, len(s.box))
    copy(cp, s.box)
    s.lock.RUnlock()
    for i := len(cp)/2 - 1; i >= 0; i-- {
        opp := len(cp) - 1 - i
        cp[i], cp[opp] = cp[opp], cp[i]
    }
    return cp
}

func (s *Storage) postMsg(id string, msg string) {
    s.lock.Lock()
    u, ok := s.users[id]
    if ok {
        s.box = append(s.box, Msg{
            FromName: u.Name,
            Time:     time.Now(),
            Content:  msg,
        })
    }
    s.lock.Unlock()
}

func (s *Storage) putStyle(id string, style string) {
    s.lock.Lock()
    u, ok := s.users[id]
    if ok {
        u.PreferredStyle = style
        s.users[id] = u
    }
    s.lock.Unlock()
}

func (s *Storage) putName(id string, name string) {
    s.lock.Lock()
    u, ok := s.users[id]
    if ok {
        s.box = append(s.box, serverMsg("User '"+u.Name+"' is now known as '"+name+"'"))
        u.Name = name
        s.users[id] = u
    }
    s.lock.Unlock()
}

func (s *Storage) getUser(id string) *User {
    s.lock.RLock()
    u, ok := s.users[id]
    s.lock.RUnlock()
        if ok {
        u.Seen <- struct{}{}
        return &u
    } else {
        return nil
    }
}

func (s *Storage) deleteWhenInactive(seen chan struct{}, id string) {
    for {
        select {
        case <-seen:
        case <-time.After(10 * time.Second):
            s.lock.Lock()
            u, ok := s.users[id]
            if ok {
                s.box = append(s.box, serverMsg("User '"+u.Name+"' has left"))
                delete(s.users, id)
            }
            s.lock.Unlock()
            return
        }
    }
}

func (s *Storage) createUser() *User {
    b := make([]byte, 20)
    _, err := rand.Read(b)
    if err != nil {
        panic(err)
    }
    u := User{
        Name:           "Guest_" + strconv.Itoa(rand.Intn(1000)),
        PrivateId:      hex.EncodeToString(b),
        Seen:           make(chan struct{}),
        PreferredStyle: "brutal",
    }
    s.lock.Lock()
    s.users[u.PrivateId] = u
    s.box = append(s.box, serverMsg("User '"+u.Name+"' has joined"))
    s.lock.Unlock()
    go s.deleteWhenInactive(u.Seen, u.PrivateId)
    return &u
}

func main() {
    http.HandleFunc("/", makeHandler())
    log.Printf("Server listening on :8080")
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal(err)
    }
}
