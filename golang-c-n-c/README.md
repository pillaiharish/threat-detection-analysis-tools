# Photo Capture and Location App

This project captures photos using the webcam and obtains the user's geolocation. It captures 5 photos within 2 seconds and uploads them to a Go server, which stores them along with the location data.

## Features

- Captures photos using the user's webcam.
- Captures 5 photos within 2 seconds.
- Renames photos sequentially (photo1.jpg, photo2.jpg, etc.).
- Obtains the user's geolocation.
- Uploads photos and location data to a Go server.
- Saves photos and location data in the server's `uploads` directory.

## Prerequisites

- Go (Golang) installed on your machine.
- OpenSSL installed on your machine.

## Setup

1. Clone the repository:
    ```bash
    git clone https://github.com/pillaiharish/threat-detection-analysis-tools.git
    cd golang-c-n-c
    ```

2. Generate self-signed SSL certificates:
    ```bash
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout server.key -out server.crt -subj "/CN=localhost"
    ```

3. Ensure the directory structure is as follows:
    ```
    golang-c-n-c/
    ├── main.go
    ├── server.crt
    ├── server.key
    ├── index.html
    └── uploads/  # Ensure this directory exists or will be created by the Go code
    ```

## Running the Server

1. Run the Go server:
    ```bash
    go run main.go
    ```

2. Open your browser and navigate to your device's Private IP from ifconfig:
    ```
    https://192.168.0.148:8443
    ```

3. The application will automatically capture 5 photos and upload them along with the location data.

## Camera and Location permission request

![Cam and location permission request](https://github.com/pillaiharish/threat-detection-analysis-tools/blob/main/golang-c-n-c/screen-capture/cam_permission_request.png)



## Image, latitude and longitue capture

![Photo and location capture](https://github.com/pillaiharish/threat-detection-analysis-tools/blob/main/golang-c-n-c/screen-capture/photo_and_location_captured_successfully.png)
