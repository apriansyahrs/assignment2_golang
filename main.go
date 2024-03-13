package main

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

type Order struct {
    ID          uint   `gorm:"primaryKey" json:"id"`
    CustomerName string `json:"customerName"`
    OrderedAt   time.Time  `json:"orderedAt"`
    Items       []Item     `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"items"`
}

type Item struct {
    ID          uint   `gorm:"primaryKey" json:"id"`
    Code        string `json:"itemCode"`
    Description string `json:"description"`
    Quantity    uint   `json:"quantity"`
    OrderID     uint   `json:"orderId"`
}

var db *gorm.DB

func main() {
    dsn := "host=127.0.0.1 user=postgres password= dbname=postgres port=5432 sslmode=disable TimeZone=Asia/Jakarta"
    var err error
    db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        panic("failed to connect database")
    }

    db.AutoMigrate(&Order{}, &Item{})

    r := gin.Default()

    r.POST("/orders", createOrder)
    r.GET("/orders", getOrders)
    r.PUT("/orders/:id", updateOrder)
    r.DELETE("/orders/:id", deleteOrder)

    r.Run(":8080")
}

func createOrder(c *gin.Context) {
    var order Order
    if err := c.BindJSON(&order); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    db.Create(&order)
    c.JSON(http.StatusCreated, order)
}

func getOrders(c *gin.Context) {
    var orders []Order
    db.Preload("Items").Find(&orders)
    c.JSON(http.StatusOK, orders)
}

func updateOrder(c *gin.Context) {
    var order Order
    id := c.Param("id")
    if err := db.Preload("Items").Where("id = ?", id).First(&order).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Record not found!"})
        return
    }

    var newOrder Order
    if err := c.BindJSON(&newOrder); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    order.CustomerName = newOrder.CustomerName
    order.OrderedAt = newOrder.OrderedAt

    for _, newItem := range newOrder.Items {
        var existingItem Item
        if err := db.Where("id = ?", newItem.ID).First(&existingItem).Error; err != nil {
            newItem.OrderID = order.ID
            db.Create(&newItem)
        } else {
            existingItem.Code = newItem.Code
            existingItem.Description = newItem.Description
            existingItem.Quantity = newItem.Quantity
            db.Save(&existingItem)
        }
    }

    c.JSON(http.StatusOK, order)
}

func deleteOrder(c *gin.Context) {
    id := c.Param("id")

    var order Order
    if err := db.Preload("Items").First(&order, id).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Record not found!"})
        return
    }

    tx := db.Begin()

    if err := tx.Unscoped().Delete(&order.Items).Error; err != nil {
        tx.Rollback()
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if err := tx.Unscoped().Delete(&order).Error; err != nil {
        tx.Rollback()
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    tx.Commit()

    c.JSON(http.StatusOK, gin.H{"message": "Success delete"})
}
