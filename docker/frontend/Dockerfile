FROM node:20-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

ARG VITE_API_URL=http://localhost:3001
ARG VITE_ELECTRIC_URL=http://localhost:3000
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_ELECTRIC_URL=$VITE_ELECTRIC_URL

RUN npm run build

EXPOSE 5173

CMD ["npm", "run", "preview", "--", "--port", "5173", "--host", "0.0.0.0"]
