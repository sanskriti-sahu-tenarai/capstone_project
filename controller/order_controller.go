package controller

import (
	"errors"
	"net/http"
	"strconv"

	"order-management/model"
	"order-management/observability"
	"order-management/service"
	"order-management/utility"

	"github.com/gin-gonic/gin"
)

type OrderController struct {
	service service.OrderService
}

func NewOrderController(s service.OrderService) *OrderController {
	return &OrderController{service: s}
}

// Login godoc
// @Summary      Authenticate user/admin
// @Description  Obtains a JWT bearer token for role 'admin' or 'user'
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body model.LoginRequest true "Login Credentials (admin:admin123 or user:user123)"
// @Success      200 {object} model.LoginResponse
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Router       /api/v1/auth/login [post]
func (ctrl *OrderController) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Username and password required"})
		return
	}

	var role string
	if req.Username == "admin" && req.Password == "admin123" {
		role = "admin"
	} else if req.Username == "user" && req.Password == "user123" {
		role = "user"
	} else {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse{Error: "Invalid username or password"})
		return
	}

	token, err := utility.GenerateToken(req.Username, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, model.LoginResponse{
		Token: token,
		Role:  role,
	})
}

// CreateOrder godoc
// @Summary      Create an Order
// @Description  Creates a new order and queries the Product Microservice for pricing
// @Tags         orders
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        order body model.CreateOrderRequest true "Create Order Request"
// @Success      201 {object} model.Order
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      500 {object} model.ErrorResponse
// @Router       /api/v1/orders [post]
func (ctrl *OrderController) CreateOrder(c *gin.Context) {
	var req model.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		observability.ErrorCount.WithLabelValues("bad_request").Inc()
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid input: " + err.Error()})
		return
	}

	order, err := ctrl.service.CreateOrder(c.Request.Context(), req)
	if err != nil {
		observability.ErrorCount.WithLabelValues("internal_error").Inc()
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Failed to create order"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetOrder godoc
// @Summary      Get Order by ID
// @Description  Retrieves details of a specific order
// @Tags         orders
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "Order ID"
// @Success      200 {object} model.Order
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Router       /api/v1/orders/{id} [get]
func (ctrl *OrderController) GetOrder(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid order ID"})
		return
	}

	order, err := ctrl.service.GetOrder(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Failed to fetch order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders godoc
// @Summary      List Orders
// @Description  Retrieves paginated orders
// @Tags         orders
// @Security     BearerAuth
// @Produce      json
// @Param        limit query int false "Limit (default 20)"
// @Param        offset query int false "Offset (default 0)"
// @Success      200 {array} model.Order
// @Failure      401 {object} model.ErrorResponse
// @Failure      500 {object} model.ErrorResponse
// @Router       /api/v1/orders [get]
func (ctrl *OrderController) ListOrders(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	orders, err := ctrl.service.ListOrders(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse{Error: "Failed to list orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// UpdateOrderStatus godoc
// @Summary      Update Order Status (Admin Only)
// @Description  Updates the status of an existing order to CREATED, CONFIRMED, or CANCELLED
// @Tags         orders
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path int true "Order ID"
// @Param        body body model.UpdateOrderStatusRequest true "Status Update Payload"
// @Success      200 {object} model.Order
// @Failure      400 {object} model.ErrorResponse
// @Failure      401 {object} model.ErrorResponse
// @Failure      403 {object} model.ErrorResponse
// @Failure      404 {object} model.ErrorResponse
// @Router       /api/v1/orders/{id}/status [put]
func (ctrl *OrderController) UpdateOrderStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid order ID"})
		return
	}

	var req model.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: "Invalid request status: " + err.Error()})
		return
	}

	updated, err := ctrl.service.UpdateOrderStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, model.ErrorResponse{Error: "Order not found"})
			return
		}
		c.JSON(http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}
