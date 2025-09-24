package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type RestCSV struct {
	ID            int64
	Name          string
	Cuisines      string
	Lat           float64
	Lng           float64
	AreaName      string
	City          string
	Region        string
	MichelinAward string
	PriceCategory string
	Image         string
	URL           string
}

func main() {
	// --- 1. 连接到数据库 ---
	// 最佳实践：从环境变量中读取数据库连接字符串
	// 这样可以避免将密码等敏感信息硬编码在代码里
	err := godotenv.Load()
	if err != nil {
		log.Println("警告: 无法加载 .env 文件。将依赖系统环境变量。")
	}
	dbUrl := os.Getenv("SUPABASE_DB_URL")
	if dbUrl == "" {
		log.Fatal("请设置 SUPABASE_DB_URL 环境变量")
	}

	// 使用 pgx.Connect 连接数据库
	// context.Background() 是一个空的上下文，用于管理连接的生命周期
	conn, err := pgx.Connect(context.Background(), "postgresql://postgres:lxp120164674@db.kmqlhcrpcznxkoryhoec.supabase.co:5432/postgres")
	if err != nil {
		log.Fatalf("无法连接到数据库: %v\n", err)
	}
	// 在 main 函数结束时，确保关闭数据库连接
	defer conn.Close(context.Background())

	fmt.Println("🎉 成功连接到 Supabase 数据库!")

	// // --- 2. 插入一条新数据 (Create) ---
	// fmt.Println("\n--> 正在插入一条新产品...")
	// var newProductID int64
	// // 使用 $1, $2 等占位符来防止 SQL 注入
	// // 使用 QueryRow(...).Scan(...) 来执行并取回由 RETURNING 返回的 id
	// err = conn.QueryRow(context.Background(),
	// 	"INSERT INTO products (name, price, category) VALUES ($1, $2, $3) RETURNING id",
	// 	"Go-powered Keyboard", 199.99, "Electronics").Scan(&newProductID)

	// if err != nil {
	// 	log.Fatalf("插入数据失败: %v\n", err)
	// }
	// fmt.Printf("✅ 成功插入新产品，ID为: %d\n", newProductID)

	// --- 3. 查询所有数据 (Read) ---
	fmt.Println("\n--> 正在查询所有产品...")
	rows, err := conn.Query(context.Background(), "SELECT id, name, cuisine, lat, lng, area, city, region, rank, price_category, image, url FROM michelin LIMIT 10")
	if err != nil {
		log.Fatalf("查询数据失败: %v\n", err)
	}
	// 确保在处理完结果后关闭 rows
	defer rows.Close()

	// 创建一个 Product 切片来存储所有查询结果
	var records []RestCSV

	// 遍历查询结果
	for rows.Next() {
		var r RestCSV
		// 将每一行的数据扫描到 Product 结构体中
		err := rows.Scan(&r.ID, &r.Name, &r.Cuisines, &r.Lat, &r.Lng, &r.AreaName, &r.City, &r.Region, &r.MichelinAward, &r.PriceCategory, &r.Image, &r.URL)
		if err != nil {
			log.Fatalf("扫描行数据失败: %v\n", err)
		}
		records = append(records, r)
	}

	// 检查遍历过程中是否有错误
	if err := rows.Err(); err != nil {
		log.Fatalf("遍历查询结果时出错: %v\n", err)
	}

	fmt.Println("✅ 查询成功，数据库中的产品有:")
	for _, p := range records {
		fmt.Printf("  - ID: %d, 名称: %s, 餐厅类型: %s, 纬度: %f, 经度: %f, 区域: %s, 城市: %s, 地区: %s, 米其林评级: %s, 价格区间: %s, 图片: %s, 链接: %s\n",
			p.ID, p.Name, p.Cuisines, p.Lat, p.Lng, p.AreaName, p.City, p.Region, p.MichelinAward, p.PriceCategory, p.Image, p.URL)
	}
}
