package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Niraj-Shaw/orderfoodonline/internal/config"
	"github.com/Niraj-Shaw/orderfoodonline/internal/models"
	"github.com/Niraj-Shaw/orderfoodonline/internal/promovalidator"
	"github.com/Niraj-Shaw/orderfoodonline/internal/repository/memory"
	"github.com/Niraj-Shaw/orderfoodonline/internal/service"
	transporthttp "github.com/Niraj-Shaw/orderfoodonline/internal/transport/http"
	"github.com/Niraj-Shaw/orderfoodonline/internal/util"
)

func main() {
	// logger
	log := util.NewLogger()

	// config
	cfg := config.Load()

	// repositories (in-memory)
	productRepo := memory.NewProductRepo(seedProducts()) // expects []models.Product
	orderRepo := memory.NewOrderRepo()

	// promo validator (case-sensitive)
	validator := promovalidator.NewValidatorService(promovalidator.Config{
		Dir:                      cfg.CouponDir,
		Files:                    []string{"couponbase1.gz", "couponbase2.gz", "couponbase3.gz"},
		MinLen:                   8,
		MaxLen:                   10,
		RequiredHits:             2,
		MaxConcurrentValidations: 2,
	})

	if err := validator.LoadCouponFiles(); err != nil {
		log.Fatalf("validator configuration error: %v", err)
	}
	log.Infof("validator configured for directory: %s (files will be scanned on-demand)", cfg.CouponDir)

	// services
	productSvc := service.NewProductService(productRepo)
	orderSvc := service.NewOrderService(productSvc, orderRepo, validator)

	// http server
	srv := transporthttp.NewServer(&cfg, productSvc, orderSvc, log)

	// start
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	log.Infof("server listening on %s", cfg.ServerAddr)
	log.Infof("health:  http://%s/healthz", cfg.ServerAddr)
	log.Infof("api:     http://%s/api", cfg.ServerAddr)

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infof("shutting down…")
	if err := srv.Stop(); err != nil {
		log.Errorf("shutdown error: %v", err)
	}
	log.Infof("bye")
}

// seedProducts returns your initial menu.
func seedProducts() []models.Product {
	return []models.Product{
		{ID: "1", Name: "Chicken Waffle", Price: 12.99, Category: "Waffle"},
		{ID: "2", Name: "Belgian Waffle", Price: 9.99, Category: "Waffle"},
		{ID: "3", Name: "Caesar Salad", Price: 8.99, Category: "Salad"},
		{ID: "4", Name: "Grilled Chicken", Price: 15.99, Category: "Main Course"},
		{ID: "5", Name: "Pasta Carbonara", Price: 13.99, Category: "Pasta"},
		{ID: "6", Name: "Chocolate Cake", Price: 6.99, Category: "Dessert"},
		{ID: "7", Name: "Coffee", Price: 3.99, Category: "Beverage"},
		{ID: "8", Name: "Orange Juice", Price: 4.99, Category: "Beverage"},
		{ID: "9", Name: "Fish Tacos", Price: 11.99, Category: "Mexican"},
		{ID: "10", Name: "Burger Deluxe", Price: 14.99, Category: "Burger"},
	}
}
