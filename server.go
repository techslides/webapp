package main

import (
    "database/sql"
    "github.com/codegangsta/martini"
    "github.com/codegangsta/martini-contrib/binding"
    "github.com/codegangsta/martini-contrib/render"
    "github.com/codegangsta/martini-contrib/sessions"
    "github.com/coopernurse/gorp"
    _ "github.com/go-sql-driver/mysql"
    "github.com/codegangsta/martini-contrib/sessionauth"
    "log"
    "net/http"
    "time"
    "html/template"
    "strconv"
    "regexp"
    "strings"
)

type Post struct {
    // db tag lets you specify the column name if it differs from the struct field
    Id      int64 `db:"post_id"`
    Created int64
    Title   string `form:"Title" binding:"required"`
    Body    string `form:"Body"`
    UserId  int64 `form:"UserId"`
    Url     string
}


type User struct {
    Id            int64  `form:"id" db:"id"`
    Email      string `form:"email" db:"email" binding:"required"`
    Password      string `form:"password" db:"password" binding:"required"`
    Name          string `form:"name" db:"name"`
    authenticated bool   `form:"-" db:"-"`
}

func newUser(email string, password string, name string, authenticated bool) User {
    return User{
        Email:   email,
        Password:    password,
        Name: name,
        authenticated:   authenticated,    
    }
}

func newPost(title string, body string, user int64) Post {

    //let's make pretty urls from title
    reg, err := regexp.Compile("[^A-Za-z0-9]+")
    if err != nil {
      log.Fatal(err)
    }
    prettyurl := reg.ReplaceAllString(title, "-")
    prettyurl = strings.ToLower(strings.Trim(prettyurl, "-"))

    return Post{
        Created: time.Now().Unix(),
        Title:   title,
        Body:    body,
        UserId:  user,
        Url: prettyurl,
    }
}


func checkErr(err error, msg string) {
    if err != nil {
        log.Fatalln(msg, err)
    }
}




//BINDING: https://github.com/codegangsta/martini-contrib/tree/master/binding
//sample custom Post struct validation
func (bp Post) Validate(errors *binding.Errors, req *http.Request) {
    //custom validation
    if len(bp.Title) == 0 {
        errors.Fields["title"] = "Title cannot be empty"
    }
}

//sample custom User struct validation
func (bp User) Validate(errors *binding.Errors, req *http.Request) {
    //custom validation
}



//SESSIONAUTH: https://github.com/codegangsta/martini-contrib/tree/master/sessionauth
// GetAnonymousUser should generate an anonymous user model for all sessions. This should be an unauthenticated 0 value struct.
func GenerateAnonymousUser() sessionauth.User {
    return &User{}
}

// Login will preform any actions that are required to make a user model officially authenticated.
func (u *User) Login() {
    // Update last login time, add to logged-in user's list
    u.authenticated = true
}

// Logout will preform any actions that are required to completely logout a user.
func (u *User) Logout() {
    // Remove from logged-in user's list
    u.authenticated = false
}

func (u *User) IsAuthenticated() bool {
    return u.authenticated
}

func (u *User) UniqueId() interface{} {
    return u.Id
}

// GetById will populate a user object from a database model with a matching id.
func (u *User) GetById(id interface{}) error {
    err := dbmap.SelectOne(u, "SELECT * FROM users WHERE id = ?", id)
    if err != nil {
       return err
    }
    return nil
}




//INITAL DATABASE SETUP
var dbmap *gorp.DbMap

func initDb() *gorp.DbMap {

    db, err := sql.Open("mysql", "USERNAME:PASSWORD@unix(/var/run/mysqld/mysqld.sock)/webapp")
    checkErr(err, "sql.Open failed")

    dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}

    dbmap.AddTableWithName(User{}, "users").SetKeys(true, "Id")

    err = dbmap.CreateTablesIfNotExists()
    checkErr(err, "Create tables failed")

    dbmap.AddTableWithName(Post{}, "posts").SetKeys(true, "Id")
    err = dbmap.CreateTablesIfNotExists()
    checkErr(err, "Create tables failed")

    return dbmap
}




func main() {
    //Change secret123 to something more secure and store session in backend instead of cookie
    store := sessions.NewCookieStore([]byte("secret123"))

    dbmap = initDb()
    
    defer dbmap.Db.Close()

    err := dbmap.TruncateTables()
    checkErr(err, "TruncateTables failed")

    u1 := newUser("bob@domain.com", "pass", "Bob", false)

    //insert rows
    err = dbmap.Insert(&u1)
    checkErr(err, "Insert failed")

    //create two posts, assign to user 1 above
    p1 := newPost("Post 1", "Lorem ipsum lorem ipsum",1)
    p2 := newPost("Post 2", "This is my second post",1)

    // insert rows
    err = dbmap.Insert(&p1, &p2)
    checkErr(err, "Insert failed")

   
    m := martini.Classic()

    m.Use(render.Renderer(render.Options{
        Directory: "templates",
        Layout: "layout",
        Funcs: []template.FuncMap{
            {
                "formatTime": func(args ...interface{}) string { 
                    t1 := time.Unix(args[0].(int64), 0)
                    return t1.Format(time.Stamp)
                },
            },
        },
    }))


    m.Use(sessions.Sessions("my_session", store))
    m.Use(sessionauth.SessionUser(GenerateAnonymousUser))

    sessionauth.RedirectUrl = "/login"
    sessionauth.RedirectParam = "next"


    //ROUTES

    m.Get("/register", func(r render.Render, user sessionauth.User) {

        //redirect to homepage if already authenticated
        if(user.IsAuthenticated()){
            r.Redirect("/")
        } else {
            r.HTML(200, "register", nil)
        }
        
    })


    m.Get("/login", func(r render.Render, user sessionauth.User) {

        //redirect to homepage if already authenticated
        if(user.IsAuthenticated()){
            r.Redirect("/")
        } else {
            r.HTML(200, "login", nil)
        }
    })


    m.Post("/login", binding.Form(User{}), func(session sessions.Session, postedUser User, r render.Render, ferr binding.Errors, req *http.Request) {

    	log.Println(ferr)

    	//Example of server side error validation for the client side form
        if ferr.Count() > 0 {
            newmap := map[string]interface{}{"metatitle":"Registration", "errormessage":"Error with Form Submission"}
            r.HTML(200, "login", newmap)
        } else {

	        user := User{}
	        
	        //check login credentails with DataBase
	        err := dbmap.SelectOne(&user, "SELECT * FROM users WHERE email = ? and password = ?", postedUser.Email, postedUser.Password)
	        if err != nil {
	            r.Redirect(sessionauth.RedirectUrl)
	            return
	        } else {
	            err := sessionauth.AuthenticateSession(session, &user)
	            if err != nil {
	                r.JSON(500, err)
	            }

	            params := req.URL.Query()
	            redirect := params.Get(sessionauth.RedirectParam)
	            r.Redirect(redirect)
	            return
	        }

	   }

    })


    m.Get("/logout", sessionauth.LoginRequired, func(session sessions.Session, user sessionauth.User, r render.Render) {
        sessionauth.Logout(session, user)
        r.Redirect("/")
    })


    m.Get("/", func(r render.Render, authuser sessionauth.User) {

        var posts []Post
        _, err = dbmap.Select(&posts, "select * from posts order by post_id")
        checkErr(err, "Select failed")

        newmap := map[string]interface{}{"metatitle": "HomePage", "authuser": authuser, "posts": posts}
        r.HTML(200, "posts", newmap)
    })


    m.Get("/users", func(r render.Render, authuser sessionauth.User) {

        var users []User
        
        _, err = dbmap.Select(&users, "select * from users order by id")
        checkErr(err, "Select failed")
            
        newmap := map[string]interface{}{"metatitle": "Users listing", "authuser": authuser, "users": users}
        r.HTML(200, "users", newmap)

    })


    m.Get("/users/:id", sessionauth.LoginRequired, func(args martini.Params, r render.Render, authuser sessionauth.User) {
        
        var user User
            
        err = dbmap.SelectOne(&user, "select * from users where id=?", args["id"])
            
        //simple error check
        if err != nil {
            newmap := map[string]interface{}{"metatitle":"404 Error", "message":"User not found"}
            r.HTML(404, "error", newmap)
        } else {

            var posts []Post
            _, err = dbmap.Select(&posts, "select * from posts where UserId=?", args["id"])
            checkErr(err, "Select failed")
       
            newmap := map[string]interface{}{"metatitle": user.Name+" profile page", "authuser": authuser, "user": user, "posts": posts}
            r.HTML(200, "user", newmap)
        } 

    })


    m.Post("/users", binding.Form(User{}), func(session sessions.Session, user User, ferr binding.Errors, r render.Render) {

    	//Example of server side error validation for the client side form
        if ferr.Count() > 0 {
            newmap := map[string]interface{}{"metatitle":"Registration", "errormessage":"Error with Form Submission"}
            r.HTML(200, "register", newmap)
        } else {

	        u := newUser(user.Email, user.Password, user.Name, user.authenticated)
	            
	        err = dbmap.Insert(&u)
	        checkErr(err, "Insert failed")

	        //create the session and redirect always to homepage
	        err := sessionauth.AuthenticateSession(session, &u)
	        if err != nil {
	           r.JSON(500, err)
	        }

	        r.Redirect("/")
        }

    })


    m.Put("/users/:id", binding.Bind(User{}), func(args martini.Params, user User, r render.Render, authuser sessionauth.User) {

        //convert string to int64 so you can match the struct (passing userid via ajax does not work as it comes in as a string)
        f, _ := strconv.ParseInt(args["id"],0,64)

        //only allow the authenticated user to update his user attributes
        if(authuser.UniqueId() == f){
            
            //specify the user id
            user.Id = f

            count, err := dbmap.Update(&user)
            checkErr(err, "Update failed")
            log.Println("Rows updated:", count)

            if count == 1 {
               newmap := map[string]interface{}{"responseText":"success"} 
               r.JSON(200, newmap)
            } else {
                newmap := map[string]interface{}{"responseText":"error"}
                r.JSON(400, newmap)
            } 
        

        } else {
            newmap := map[string]interface{}{"responseText":"You are not allowed to update this resource."}
            r.JSON(403, newmap)              
        }

    })


    m.Delete("/users/:id", func(args martini.Params, r render.Render, authuser sessionauth.User) {
        
        //convert id from string to int64
        f, _ := strconv.ParseInt(args["id"],0,64)

        //only allow the authenticated user to delete him or her
        if(authuser.UniqueId() == f){

            _, err = dbmap.Exec("delete from users where id=?", args["id"])
            checkErr(err, "Delete failed")

            if err == nil {
               newmap := map[string]interface{}{"responseText":"success"}   
               r.JSON(200, newmap)
               //if you delete yourself, Ajax should redirec you
            } else {
                newmap := map[string]interface{}{"responseText":"error"}
                r.JSON(400, newmap)
            } 


        } else {
            newmap := map[string]interface{}{"responseText":"You are not allowed to delete this resource."}
            r.JSON(403, newmap)              
        }

    })


    m.Post("/posts", sessionauth.LoginRequired, binding.Bind(Post{}), func(post Post, r render.Render, authuser sessionauth.User) {

        //convert to int64
        f := authuser.UniqueId().(int64)
        p1 := newPost(post.Title, post.Body,f)

        err = dbmap.Insert(&p1)
        checkErr(err, "Insert failed")

        r.Redirect("/")
    })


    m.Get("/posts/:id", func(args martini.Params, r render.Render, authuser sessionauth.User) {

        var post Post

        err = dbmap.SelectOne(&post, "select * from posts where post_id=?", args["id"])
        
        //simple error check
        if err != nil {
          newmap := map[string]interface{}{"metatitle":"404 Error", "message":"This is not found"}
          r.HTML(404, "error", newmap)
        } else {
          newmap := map[string]interface{}{"metatitle": post.Title+" more custom","authuser": authuser, "post": post}
          r.HTML(200, "post", newmap)
        }
    })


    m.Get("/p/:str", func(args martini.Params, r render.Render, authuser sessionauth.User) {

        var post Post

        err = dbmap.SelectOne(&post, "select * from posts where url=?", args["str"])
        
        //simple error check
        if err != nil {
          newmap := map[string]interface{}{"metatitle":"404 Error", "message":"This is not found"}
          r.HTML(404, "error", newmap)
        } else {
          newmap := map[string]interface{}{"metatitle": post.Title+" more custom","authuser": authuser, "post": post}
          r.HTML(200, "post", newmap)
        }
    })


    m.Put("/posts/:id", binding.Bind(Post{}), func(args martini.Params, post Post, r render.Render, authuser sessionauth.User) {

        var newTitle = post.Title
        var newBody = post.Body

        err = dbmap.SelectOne(&post, "select * from posts where post_id=?", args["id"])

        //simple database error check
        if err != nil {
          newmap := map[string]interface{}{"message":"Something went wrong."}
          r.JSON(400, newmap)
        } else {

          //owner check
          if(authuser.UniqueId() == post.UserId){

            post.Title=newTitle
            post.Body=newBody

            count, err := dbmap.Update(&post)
            checkErr(err, "Update failed")
            
            if count == 1 {
               newmap := map[string]interface{}{"responseText":"success"}
               r.JSON(200, newmap)
            } else {
               newmap := map[string]interface{}{"responseText":"error"}
               r.JSON(400, newmap)
            }  

          } else {
            newmap := map[string]interface{}{"responseText":"You are not allowed to modify this resource."}
            r.JSON(403, newmap)
          }

        } 
        
    })


    m.Delete("/posts/:id", func(args martini.Params, r render.Render, authuser sessionauth.User) {

        //retrieve the post to check the real owner
        var post Post
        err = dbmap.SelectOne(&post, "select * from posts where post_id=?", args["id"])

        //simple DB error check
        if err != nil {
          newmap := map[string]interface{}{"message":"Something went wrong."}
          r.JSON(400, newmap)
        } else {

          //owner check
          if(authuser.UniqueId() == post.UserId){

            //delete it
            _, err := dbmap.Delete(&post)
            checkErr(err, "Delete failed")

            newmap := map[string]interface{}{"responseText":"success"}
            r.JSON(200, newmap)

          } else {
            newmap := map[string]interface{}{"responseText":"You are not allowed to delete this resource."}
            r.JSON(403, newmap)              
          }

       }


    })

    m.Run()
}
