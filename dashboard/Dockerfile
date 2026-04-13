FROM node:22-alpine
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
EXPOSE 5173
ENV VITE_API_TARGET=http://api:3000
CMD ["npm", "run", "dev"]
