# Go Geocache API

A Go 1.20 API server for storing and retrieving Geocaches.  The data stored for each geocache is:
- Id
- Name
- Latitude
- Longitude
- Tags (set of strings)

The "backend" will generate an auto-incrementing `uint64` `iD` for each geocache.

## Endpoints

> A complete implementation would include an OpenAPI 3.0 specification instead of listing the endpoints here (see [ToDos](#todos) #1 for details).
>
> All of the following curl examples assume that you have started the server listening on `localhost:8080`.

There is some test data in the `integration_test/test_data/` directory that can be used to exercise the API.

All endpoints are accessible via the following prefix `http://<host>:<port>/v1/`.

- **POST geocache record**
    ```
    geocaches
    ```
    With the following JSON
    ```
    {
      "name": "string",
      "lat": float,
      "long": float,
      "tags": ["string"]
    }
    ```
    ```
    curl -X POST http://localhost:8080/v1/geocaches -d @create-geocache-oregon.json
    ```

- **GET geocache by name**
    ```
    geocaches/<name>
    ```
    Will return
    ```
    {
      "id": 1,
      "name": "s1",
      "lat": 38.394432064782755,
      "long": -75.0613367366317,
      "tags": [
        "atlantic",
        "flowrate",
        "ocean"
      ]
    }
    ```
    ```
    curl -X GET http://localhost:8080/v1/geocaches/oregon
    ```

- **GET geocaches by tag** will return an array of geocaches including all geocaches that have at least one of the tags provided in the query.
    ```
    geocaches?tags=string,string,...
    ```
    Will return
    ```
    [
      {
        "id": 1,
        "name": "s1",
        "lat": 38.394432064782755,
        "long": -75.0613367366317,
        "tags": [
          "atlantic",
          "flowrate",
          "ocean"
        ]
      },
      {
        "id": 2,
        "name": "s2",
        "lat": 39.394432064782755,
        "long": -74.0613367366317,
        "tags": [
          "ocean"
        ]
      },
    ]
    ```
    ```
    curl -X GET http://localhost:8080/v1/geocaches?tags=ocean,s
    ```

- **PUT geocaches by name** to update the geocache's metadata
    ```
    geocaches/<name>
    ```
    With the following JSON
    ```
    {
      "lat": float,
      "long": float,
      "tags": ["string"]
    }
    ```
    Will return
    ```
    {
      "name": "string",
      "lat": float,
      "long": float,
      "tags": ["string"]
    }
    ```
    ```
    curl -X PUT http://localhost:8080/v1/geocaches/australia -d @create-geocache-australia-update.json
    ```

- **GET geocaches nearest to a given lat/long** will return an array of geocaches that are nearest the provided gps coordinates.  Currently, this does not implement distance or a limit and will just return the set of geocaches that share the same quadrant as the gps coordinate requested.  The `GeoStore` type is implemented with a `InMemGeoStore` which is simply a quadtree that stores each geocache in the correct gps quadrant.  An actual production implementation would utilize a more robust, distributed, GeoLocation specific datastore and caching layer.
    ```
    geocaches/nearest?lat=<float>&long=<float>&maxdistance=<int>&limit=<int>
    ```
    Will return an array of the nearest geocaches
    ```
    [
      {
        "id": 1,
        "name": "s1",
        "lat": 38.394432064782755,
        "long": -75.0613367366317,
        "tags": [
          "atlantic",
          "flowrate",
          "ocean"
        ]
      },
      {
        "id": 2,
        "name": "s2",
        "lat": 39.394432064782755,
        "long": -74.0613367366317,
        "tags": [
          "ocean"
        ]
      },
    ]
    ```
    ```
    curl -X GET "http://localhost:8080/v1/geocaches/nearest?lat=-23.0&long=124.923&maxdistance=0&limit=-1"
    ```

### ToDos

The following are a list of things that I would tackle were this an actual piece of software that I was going to run in production.

1. **Implement updating the lat/long of a node in the GeoStore**:  Currently, data in the GeoStore is just added, PUTting an update to a geocache's lat long does NOT update it in the GeoStore.
1. **Implementing distance calculations and limits for the `/nearest` endpoint**: Implement a BFS from the search nexus that will, given a maximum distance, search for the nearest nodes.
1. **Generate an OpenAPI spec and Swagger documentation**:  There are number of approaches for utilizing OpenAPI for a project.  I tend to build software with the least amount of tight couplings and the control to implement the code as I see fit.  As a result, I would not build the spec and then generate the code from it because I am then tied to the specific implementations and dependencies that the code generation tool provides.  Instead I would use `[swaggo/swag](https://github.com/swaggo/swag)`, annotate the code and generate the OpenAPI 2.0 spec from the code itself.  From there, the spec can be converted to OpenAPI 3.0 and from there Swagger documentation an be generated.  This approach enables me to structure and implement the code in any way that I see fit.  The source code being the source code of truth for the OpenAPI documentation, and not the other way around.  For now, this is left out, but could be added.
1. **Implement authentication and authorization**:  The code is stubbed out for both `authn` and `authz`.  Right now I have left out the exercise of salting, hashing and storing passwords and granting and passing around auth tokens, along with defining RBAC rules and controls.
1. **TLS**: An production API should utilize `https`.
1. **Pagination and Limits**

## Running

To run the API server run the following in the repository root
```
go run ./ --port 8080
```

You can then use `curl` or PostMan or any other REST client to exercise the API endpoints.
## Running tests
Run the following to execute the unit and integration tests.  Omit setting `INTEGRATION` environment variable to only run the unit tests.
```
INTEGRATION=1 go test -v -count=1 ./...
```
The `geostore/geostore_test.TestFindNearest` will generate a PNG image of the map generated by the QuadTree in this test, written to `/var/tmp/geocache-api-map.png`

The grey lines are the boundaries of the QuadTree nested structures and the black pixels are the gps coordinates that were stored in the GeoStore during the test.

## Development
### Mocks
This project utilizes the `https://github.com/golang/mock` library.  If changes are made to any of the interfaces in the project do the following to regenerate the mocks.
1. Follow the documentation in the `gomock` [`README.md`](https://github.com/golang/mock) for installing go mock if you have not already.
1. From the root of the repository run the following to (re)generate the mocks into the /mocks dir
    ```
    go generate ./...
    ```

## Research

### GPS Datastore Data Structures
- [Quadtree](https://en.wikipedia.org/wiki/Quadtree)
- [Geohash](https://en.wikipedia.org/wiki/Geohash)
- [Understanding usage of quadtree to store location data](https://softwareengineering.stackexchange.com/questions/427008/understanding-usage-of-quadtree-to-store-location-data)
- [Improved Location Caching with Quadtrees](https://engblog.yext.com/post/geolocation-caching)
- [Design Yelp or Nearby Friends](https://learnsystemdesign.blogspot.com/p/design-yelp-or-nearby-friends.html)
- [Latitude, Longitude and Coordinate System Grids](https://gisgeography.com/latitude-longitude-coordinates/)