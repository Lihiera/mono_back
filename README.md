# Mono Back

Mono Back is the backend API for [Monobib](https://github.com/Lihiera/monobib), a restaurant discovery application using Tabelog and Michelin data.

It is built with Go, Gin, and pgx. Restaurant data is stored in Supabase PostgreSQL, while the API is deployed on Render with a shared database connection pool, request timeouts, and health checks.

## Deployment

- API: `https://mono-back.onrender.com`
- Health check: [https://mono-back.onrender.com/healthz](https://mono-back.onrender.com/healthz)
- Database readiness: [https://mono-back.onrender.com/readyz](https://mono-back.onrender.com/readyz)
- Data crawler: [Restaurants_detials_crawl](https://github.com/Lihiera/Restaurants_detials_crawl)

## API

| Method | Endpoint | Description |
| --- | --- | --- |
| `POST` | `/result?region={region}&source={source}&page={page}` | Returns up to 10 restaurants for a page |
| `POST` | `/metadata?region={region}&source={source}` | Returns restaurant map data and the total count |
| `GET` | `/healthz` | Checks whether the API process is running |
| `GET` | `/readyz` | Checks whether the database is available |

`source` must be either `tabelog` or `michelin`. `page` is zero-based.

Example:

```bash
curl -X POST "https://mono-back.onrender.com/result?region=%E6%9D%B1%E4%BA%AC%E9%83%BD&source=tabelog&page=0"
```
