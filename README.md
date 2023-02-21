# api.sonurai.com

## Running locally
To run the API locally, you'll need to have `gcloud` installed and configured.

Once you do, run `gcloud auth application-default login` to set up the credentials.

Next, set the `PROJECT_ID` and `PORT` environment variables.

Finally, start the server with `go run ./cmd/app/`.

## Image updater

The image updater function retrieves wallpapers from the Bing Wallpapers API for different countries. To be added to the database (Firestore), the wallpapers must have both 1920x1080 and 1920x1200 images available and not be duplicates. next, the images are annotated using the Google Vision API. This allows wallpapers to be searched by these labels in the future and provide better SEO.

For non-English countries, the descriptions are translated to English using the Google Translate API.

If a previously stored wallpaper appears again but with a new English description, the description is replaced but the original date is kept. The function also splits the description to retrieve the title and copyright information.

There is also splitting of the description to get the title and copyright.

