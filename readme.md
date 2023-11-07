
# Google Cloud Function for Image Shape Detection

This Google Cloud Function processes an image, calculates its shape and orientation, and saves the metadata in a `.img.meta` file in the same Google Cloud Storage bucket. The function receives an HTTP request containing the filename and the bucket where the image is located. It then performs image processing and metadata storage. The response includes the image's shape, orientation, width, and height.

## Execute locally

```bash
FUNCTION_TARGET=TransferFile LOCAL_ONLY=true go run cmd/main.go 
```

## Requirements

- Google Cloud Storage
- [Google Cloud Storage Client Library for Go](https://pkg.go.dev/cloud.google.com/go/storage)
- [MessagePack Library for Go](https://pkg.go.dev/github.com/vmihailenco/msgpack)
- [Imaging Library for Go](https://pkg.go.dev/github.com/disintegration/imaging)

## Setup

1. Set up your Google Cloud Project and enable Google Cloud Functions.

2. Deploy this function using `gcloud`:

   ```shell
   gcloud functions deploy ImageMeta \
       --runtime go121 \
       --trigger-http \
       --allow-unauthenticated
