# Spring Boot Example

This example combines Java, Spring Boot, and MySQL to expose a simple REST API. This example is based on the official [Spring Boot Documentation](https://spring.io/guides/gs/accessing-data-mysql/).

## How to Run

1. Install [Devbox](https://www.jetpack.io/devbox/docs/installing_devbox/)

1. Prepare the database by running `devbox run setup_db`. This will create the user and database that Spring expects in `stacks/spring/src/main/resources/application.properties`
1. You can now start the Spring Boot service by running `devbox run bootRun`. This will start your MySQL service and run the application
1. You can test the service using `GET localhost:8080/demo/all` or `POST localhost:8080/demo/add`. See the Spring Documentation for more details.
