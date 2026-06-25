# BrickIntersect
Enter a list of Lego peices and their colors and get a list of all sets that contain thoes pieces. Simplifying the process of finding sets in an unsorted Lego pile

## API key
A Rebrickable API key is needed to use this program, [see docs](https://rebrickable.com/api/).

The API key should be listed in `.env` file in the root of the repo as such:
```
REBRICKABLE_API_KEY=<key>
```
Or export it as an eviroment variable before running the program.

## Build and Run
### Natively:
```
export REBRICKABLE_API_KEY=<key>
go run main.go
```

### Docker:
```
sudo docker build -t brick-intersect:latest .
sudo docker run --rm -d --name=BrickIntersect -e REBRICKABLE_API_KEY=<key> -p 8090:8090 brick-intersect:latest
```
