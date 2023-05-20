# VRStore

This application receives CSV file every 30 minutes and stores its content in the database. 
The application also provides an HTTP endpoint to query the data by id.

## Getting Started

### Requirements
+ [Docker](https://docs.docker.com/)
+ [Docker Compose](https://docs.docker.com/compose/install/)


### Clone the repository

```git clone https://github.com/avag-sargsyan/vrstore.git```

### Run the application

```docker compose up -d```

### Get record

```curl http://localhost:1321/{id}```

E.g. ```curl http://localhost:1321/promotions/0b90f504-17ae-4b54-9f2a-9e60f86a0762```

should return: 

```{"id":"0b90f504-17ae-4b54-9f2a-9e60f86a0762","price":7.725578,"expiration_date":"2018-09-27T02:07:46Z"}```

## Additional Considerations

> The .csv file could be very big (billions of entries) - how would your application
perform?
 
The application file reads in chunks. 
No matter how big the file is, it won't be loaded into memory. 
The application will process the file in chunks and store the data in the database each chunk in a separate goroutine.
In order to make DB insertions more efficient, the application uses bulk insertions.


> Every new file is immutable, that is, you should erase and write the whole storage.

The application processes promotions.csv file located in promotions directory every 30 minutes.


> How would your application perform in peak periods (millions of requests per
minute)?

In order to handle high load it would be better to deploy the application using a container orchestration tool (Kubernetes),
scale the application horizontally behind a load balancer and implement a caching mechanism (Redis) to reduce the load on the database.

> How would you operate this app in production (e.g. deployment, scaling, monitoring)?

The application can be deployed using a container orchestration tool (Kubernetes) and scaled horizontally behind a load balancer.
The application can be monitored using Prometheus and Grafana.
Should be implemented CI/CD pipeline, backups and disaster recovery mechanisms.

## How can the application be improved?

- Restructure the application, move the business logic to a separate packages like services, handlers, database, etc.
- Add unit tests
- Instead of reading the file from the local file system, it would be better to create a service that will receive the file
- Add integration tests
- Use https://github.com/golang-migrate/migrate library to manage database migrations
- Add more logging
- Add more documentation
- Add more configuration options
- Add performance tests
- Implement caching
- Implement Kubernetes deployment
- Add CI/CD pipeline