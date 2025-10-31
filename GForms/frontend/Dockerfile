# Stage 1: Build the Angular application
FROM node:22-alpine AS build

# Define a build argument for the configuration environment
# Default value is 'production'
ARG BUILD_ENV=production

WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
# Use the build argument in the build command
RUN npm run build -- --configuration ${BUILD_ENV}

# Stage 2: Serve the application using Nginx
FROM nginx:alpine
# Copy the build output from the build stage
# Make sure this path matches the output path defined in your angular.json for the specified configuration
COPY --from=build /app/dist/gforms-app/browser /usr/share/nginx/html

COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]