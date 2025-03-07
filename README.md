# Solution by Diego Karabin

## Problem
Build an API that fetch from https://randomuser.me/api/ 15000 records of random users.
Fetch gender, first name, last name, email, city, country, and uuid for each user.
The endpoint should take up to 2.5s to respond with the required information.


## Requirements
Golang
Gin
Go-redis
Redis

To run redis with docker use:
```bash
docker run -p 6379:6379 --name redis -d redis
```

To run project:
```bash
go run . # inside the folder
```

## Video demonstration
https://www.loom.com/share/beca5d830ee7454da61b0496d2c2be16
