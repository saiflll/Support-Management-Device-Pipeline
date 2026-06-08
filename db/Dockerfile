FROM postgres:13

# Copy script SQL ke folder khusus inisialisasi Postgres
COPY init-rbac.sql /docker-entrypoint-initdb.d/

# Set timezone agar sinkron
ENV TZ=Asia/Jakarta
ENV PGTZ=Asia/Jakarta
