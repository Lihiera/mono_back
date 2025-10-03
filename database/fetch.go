package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
)

// The types for fetching data from database
type DataFetcher interface {
	GetDataFromDatabase(ctx context.Context, conn *pgx.Conn, region string, page int) ([]Item, error)
}
type TabeFetcher []TabeDB
type MiFetcher []MiDB

// The types corresponding to the database's items
type TabeDB struct {
	ID         int64
	Name       string
	Cuisines   string
	Lat        float64
	Lng        float64
	CityArea   string
	Region     string
	Rank       float64
	PriceRange pgtype.Range[pgtype.Numeric]
	Image      string
	URL        string
}

type MiDB struct {
	ID        int64
	Name      string
	Cuisines  string
	Lat       float64
	Lng       float64
	Area      string
	Region    string
	Rank      string
	PriceRank string
	Image     string
	URL       string
}

// The type for fetching metadata
type MetaDB struct {
	ID       int64
	Name     string
	Cuisines string
	Lat      float64
	Lng      float64
	Image    string
	URL      string
}

// The type for responding to the frontend
type MetaDTO struct {
	Data  []MetaItem `json:"data"`
	Count int        `json:"count"`
}

type MetaItem struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Cuisines []string `json:"cuisines"`
	Lat      string   `json:"lat"`
	Lng      string   `json:"lng"`
	Image    string   `json:"img"`
	URL      string   `json:"url"`
}

type Item struct {
	ID         int64    `json:"id"`
	Name       string   `json:"name"`
	Cuisines   []string `json:"cuisines"`
	Lat        string   `json:"lat"`
	Lng        string   `json:"lng"`
	CityArea   string   `json:"city_area"`
	Region     string   `json:"region"`
	Rank       string   `json:"rank"`
	PriceRange string   `json:"price_range"`
	Image      string   `json:"img"`
	URL        string   `json:"url"`
}

// Implementation the fetching interface
func (f *TabeFetcher) GetDataFromDatabase(ctx context.Context, conn *pgx.Conn, region string, page int) ([]Item, error) {
	offset := 10 * page
	sql := `SELECT id, name, cuisine, lat, lng, city_area, region, rank, price_range, image, url FROM tabelog WHERE region=$1 ORDER BY id OFFSET $2 LIMIT 10`
	rows, err := conn.Query(ctx, sql, region, offset)
	if err != nil {
		return nil, fmt.Errorf(": Failed to query data: %w\n", err)
	}
	defer rows.Close()

	var res []Item

	for rows.Next() {
		var rTabe TabeDB
		err := rows.Scan(&rTabe.ID, &rTabe.Name, &rTabe.Cuisines, &rTabe.Lat, &rTabe.Lng, &rTabe.CityArea, &rTabe.Region, &rTabe.Rank, &rTabe.PriceRange, &rTabe.Image, &rTabe.URL)

		if err != nil {
			return nil, fmt.Errorf("Failed to scan data: %w\n", err)
		}
		*f = append(*f, rTabe)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error traversing query results: %w\n", err)
	}

	fmt.Println("✅ Query successful:")
	for _, r := range *f {
		res = append(res, r.ToDTO())
	}
	return res, nil
}

func (f *MiFetcher) GetDataFromDatabase(ctx context.Context, conn *pgx.Conn, region string, page int) ([]Item, error) {
	offset := 10 * page
	sql := `SELECT id, name, cuisine, lat, lng, area, region, rank, price_category, image, url FROM michelin WHERE region=$1 ORDER BY id OFFSET $2 LIMIT 10`
	rows, err := conn.Query(ctx, sql, region, offset)
	if err != nil {
		return nil, fmt.Errorf(": Failed to query data: %w\n", err)
	}
	defer rows.Close()

	var res []Item

	for rows.Next() {
		var rMi MiDB
		err = rows.Scan(&rMi.ID, &rMi.Name, &rMi.Cuisines, &rMi.Lat, &rMi.Lng, &rMi.Area, &rMi.Region, &rMi.Rank, &rMi.PriceRank, &rMi.Image, &rMi.URL)

		if err != nil {
			return nil, fmt.Errorf("Failed to scan data: %w\n", err)
		}
		*f = append(*f, rMi)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Error traversing query results: %w\n", err)
	}

	fmt.Println("✅ Query successful:")
	for _, r := range *f {
		res = append(res, r.ToDTO())
	}
	return res, nil
}

func MakeFetcher(source string) DataFetcher {

	if source == "tabelog" {
		var res = TabeFetcher(make([]TabeDB, 0))
		return &res
	} else if source == "michelin" {
		var res = MiFetcher(make([]MiDB, 0))
		return &res
	}
	return nil
}

func FetchPageData(ctx context.Context, region string, page int, source string) []Item {
	err := godotenv.Load()
	if err != nil {
		log.Println("cannot load .env file")
	}
	dbUrl := os.Getenv("SUPABASE_DB_URL")
	if dbUrl == "" {
		log.Fatal("please set env variable")
	}
	config, err := pgx.ParseConfig(dbUrl)
	if err != nil {
		log.Fatal(err)
	}
	config.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		log.Fatalf("fail to connect to the database: %v\n", err)
	}
	defer conn.Close(context.Background())

	fmt.Println("🎉 success on connecting to the database!")

	fmt.Printf("\n--> Searching for %s data... in %s database", region, source)
	fetcher := MakeFetcher(source)
	res, err := fetcher.GetDataFromDatabase(ctx, conn, region, page)
	if err != nil {
		log.Fatal(err)
	}
	return res
}

func FetchMetaData(ctx context.Context, region string, source string) MetaDTO {
	err := godotenv.Load()
	if err != nil {
		log.Println("cannot load .env file")
	}
	dbUrl := os.Getenv("SUPABASE_DB_URL")
	if dbUrl == "" {
		log.Fatal("please set env variable")
	}

	conn, err := pgx.Connect(ctx, dbUrl)
	if err != nil {
		log.Fatalf("fail to connect to the database: %v\n", err)
	}
	defer conn.Close(context.Background())

	fmt.Println("🎉 success on connecting to the database!")

	fmt.Printf("\n--> Searching for %s metadata... in %s database", region, source)
	var totalCount int
	var countsql string
	var locsql string
	if source == "tabelog" {
		countsql = `SELECT COUNT(*) FROM tabelog WHERE region=$1`
		locsql = `SELECT id, name, cuisine, lat, lng, image, url FROM tabelog WHERE region=$1`
	} else {
		countsql = `SELECT COUNT(*) FROM michelin WHERE region=$1`
		locsql = `SELECT id, name, cuisine, lat, lng, image, url FROM michelin WHERE region=$1`
	}
	rows, err := conn.Query(ctx, locsql, region)
	if err != nil {
		log.Fatalf(": Failed to query data: %v\n", err)
	}
	defer rows.Close()

	var records []MetaDB
	var res []MetaItem

	for rows.Next() {
		var r MetaDB
		err := rows.Scan(&r.ID, &r.Name, &r.Cuisines, &r.Lat, &r.Lng, &r.Image, &r.URL)
		if err != nil {
			log.Fatalf("Failed to scan data: %v\n", err)
		}
		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error traversing query results: %v\n", err)
	}
	if err := conn.QueryRow(ctx, countsql, region).Scan(&totalCount); err != nil {
		log.Fatalf("Error traversing query results: %v\n", err)
	}

	fmt.Println("✅ Query successful:")
	for _, r := range records {
		res = append(res, r.ToDTO())
	}
	return MetaDTO{Data: res, Count: totalCount}
}

func rangeTOString(p pgtype.Range[pgtype.Numeric]) string {
	res := "No Price"
	low := ""
	up := ""
	if !p.Valid {
		return res
	}
	if p.LowerType != pgtype.Unbounded {
		val, err := p.Lower.Value()
		if err == nil {
			low = val.(string)
		}
	}
	if p.UpperType != pgtype.Unbounded {
		val, err := p.Upper.Value()
		if err == nil {
			up = val.(string)
		}
	}
	return fmt.Sprintf("¥%s ~ %s", low, up)
}

func (r *TabeDB) ToDTO() Item {
	var res Item
	res.ID = r.ID
	res.Name = r.Name
	res.Cuisines = strings.Split(r.Cuisines, "、")
	res.Lat = fmt.Sprint(r.Lat)
	res.Lng = fmt.Sprint(r.Lng)
	res.CityArea = r.CityArea
	res.Region = r.Region
	res.Rank = fmt.Sprintf("%.2f", r.Rank)
	res.PriceRange = rangeTOString(r.PriceRange)
	res.Image = r.Image
	res.URL = r.URL
	return res
}

func (r *MiDB) ToDTO() Item {
	var res Item
	res.ID = r.ID
	res.Name = r.Name
	res.Cuisines = strings.Split(r.Cuisines, "、")
	res.Lat = fmt.Sprint(r.Lat)
	res.Lng = fmt.Sprint(r.Lng)
	res.CityArea = r.Area
	res.Region = r.Region
	res.Rank = r.Rank
	res.PriceRange = r.PriceRank
	res.Image = r.Image
	res.URL = "https://guide.michelin.com" + r.URL
	return res
}

func (r *MetaDB) ToDTO() MetaItem {
	var res MetaItem
	res.ID = r.ID
	res.Name = r.Name
	res.Cuisines = strings.Split(r.Cuisines, "、")
	res.Lat = fmt.Sprint(r.Lat)
	res.Lng = fmt.Sprint(r.Lng)
	res.Image = r.Image
	res.URL = r.URL
	return res
}
