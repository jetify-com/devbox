# Spring Boot Example

This example combines Java, Spring Boot, and MySQL to expose a simple REST API. This example is based on the official [Spring Boot Documentation](https://spring.io/guides/gs/accessing-data-mysql/).

## How to Run

1. Install [Devbox](https://www.jetify.com/docs/devbox/installing-devbox/index)

1. Prepare the database by running `devbox run setup_db`. This will create the user and database that Spring expects in `stacks/spring/src/main/resources/application.properties`
1. You can now start the Spring Boot service by running `devbox run bootRun`. This will start your MySQL service and run the application
1. You can test the service using `GET localhost:8080/demo/all` or `POST localhost:8080/demo/add`. See the Spring Documentation for more details.

## How to Recreate this Example

1. Create a blank Devbox project with `devbox init`
2. Add the required packages with `devbox add jdk@17 mysql@latest gradle@latest`
3. Create a new Spring Boot application using the [Spring Boot initializer](https://start.spring.io/).
4. Copy the devbox.json and devbox.lock files into the project directory.
5. Initialize your mysql database by running `devbox services up`, and create the example DB and user using the `setup_db.sql` file in this directory.

## Notes

- This example uses the [Spring Boot initializer](https://start.spring.io/) to create the project. You can use any method you like to create your Spring Boot project, but you will need to make sure that the `devbox.json` and `devbox.lock` files are in the same directory as your `build.gradle` file.
- This example hardcodes a username and password for development purposes. For production or more secure usecases, you should change them and exclude them from source control.
- This distribution uses the OpenJDK. You can find other JDK distributions using `devbox search`
