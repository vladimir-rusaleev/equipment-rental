package httpapi

var routes = []string{
	"POST /api/v1/auth/register",
	"POST /api/v1/auth/login",
	"GET /api/v1/models",
	"GET /api/v1/models/{id}",
	"GET /api/v1/models/{id}/calendar",
	"GET /api/v1/me",
	"POST /api/v1/models",
	"PUT /api/v1/models/{id}",
	"GET /api/v1/models/{id}/units",
	"POST /api/v1/models/{id}/units",
	"PATCH /api/v1/units/{id}",
	"PATCH /api/v1/users/{id}/role",
	"GET /api/v1/cart",
	"PUT /api/v1/cart",
	"POST /api/v1/orders",
	"GET /api/v1/orders",
	"GET /api/v1/orders/{id}",
	"POST /api/v1/orders/{id}/cancel",
	"POST /api/v1/orders/{id}/issue",
	"POST /api/v1/orders/{id}/return",
}
