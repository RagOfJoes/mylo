FROM postgres:alpine
RUN apk update && apk upgrade
COPY initdb.sh /docker-entrypoint-initdb.d/
CMD ["docker-entrypoint.sh", "postgres"]
