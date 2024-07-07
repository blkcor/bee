package main

import (
	"bee"
	"net/http"
)

func main() {
	r := bee.New()
	r.GET("/", func(c *bee.Context) {
		c.HTML(http.StatusOK, "<h1>Hello Gee</h1>")
	})

	r.GET("/hello", func(c *bee.Context) {
		// expect /hello?name=geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Query("name"), c.Path)
	})

	r.GET("/hello/:name", func(c *bee.Context) {
		// expect /hello/geektutu
		c.String(http.StatusOK, "hello %s, you're at %s\n", c.Param("name"), c.Path)
	})

	r.GET("/assets/*filepath", func(c *bee.Context) {
		c.JSON(http.StatusOK, bee.H{"filepath": c.Param("filepath")})
	})
	v1 := r.Group("/v1")
	{
		v1.GET("/hello", func(context *bee.Context) {
			context.JSON(http.StatusOK, bee.H{
				"message": "hello",
			})
		})
	}
	r.Run(":9999")
}
