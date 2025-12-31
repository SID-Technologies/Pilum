FROM eclipse-temurin:21-jdk-alpine AS builder

WORKDIR /app
COPY .mvn .mvn
COPY mvnw pom.xml ./
RUN ./mvnw dependency:go-offline

COPY src ./src
RUN ./mvnw package -DskipTests

FROM eclipse-temurin:21-jre-alpine
COPY --from=builder /app/target/*.jar /app.jar

EXPOSE 8080
CMD ["java", "-jar", "/app.jar"]
