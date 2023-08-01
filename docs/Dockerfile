# syntax=docker/dockerfile:1 
FROM nginx:stable-alpine as deploy
WORKDIR /app
EXPOSE 80
COPY ./app/build/devbox/docs ./static/devbox/docs/
COPY ./nginx.conf /etc/nginx/nginx.conf
