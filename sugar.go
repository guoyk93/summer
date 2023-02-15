package summer

import "errors"

// Bind a generic version of [Context.Bind]
//
// example:
//
//		func actionValidate(c summer.Context) {
//			args := summer.Bind[struct {
//	       		Tenant       string `json:"header_x_tenant"`
//				Username     string `json:"username"`
//				Age 		 int    `json:"age,string"`
//			}](c)
//	        _ = args.Tenant
//	        _ = args.Username
//	        _ = args.Age
//		}
func Bind[T any](c Context) (o T) {
	c.Bind(&o)
	return
}

// Panic panic a simple error with http status code
func Panic(s string, code int) {
	panic(ErrorWithHTTPStatus(errors.New(s), code))
}
