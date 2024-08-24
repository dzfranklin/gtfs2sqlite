## Usage

```bash
> go install github.com/dzfranklin/gtfs2sqlite/cmd@latest
> gtfs2sqlite --import input.gtfs.zip --force-valid --out timetable.db
> gtfs2sqlite --export timetable.db
```

You can also clip to a geojson feature. Trips entirely outside the clip feature will be removed.

```bash
> gtfs2sqlite --export timetable.db --clip scotland-geojson.json
```
