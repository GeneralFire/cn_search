FROM node:24-alpine
WORKDIR /app

# Зависимости — только из публичного registry.npmjs.org (см. .npmrc).
# Ставим их в образ; исходники монтируются томом из docker-compose.
COPY package.json package-lock.json .npmrc ./
RUN npm ci

COPY . .

EXPOSE 7173
CMD ["npm", "run", "dev"]
