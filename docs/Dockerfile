# syntax=docker/dockerfile:1 
FROM nginx:stable-alpine as deploy
WORKDIR /app
EXPOSE 80
COPY ./app/build ./static/devbox/
COPY ./nginx.conf /etc/nginx/nginx.conf
