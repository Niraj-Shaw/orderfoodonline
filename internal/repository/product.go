package repository

import "github.com/Niraj-Shaw/orderfoodonline/internal/models"

type ProductRepository interface {
	FindAll() ([]models.Product, error)
	FindByID(id string) (*models.Product, error)
}
