package gtfs2sqlite

import (
	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"fmt"
	"github.com/tidwall/geojson"
	"github.com/tidwall/geojson/geometry"
	"log/slog"
	"strconv"
)

func Clip(inputPath string, outputPath string, clipFeature string) error {
	feature, err := geojson.Parse(clipFeature, &geojson.ParseOptions{RequireValid: true})
	if err != nil {
		return fmt.Errorf("parse clip feature: %w", err)
	}

	slog.Info(fmt.Sprintf("Writing a clipped copy of %s to %s (clipFeature has %d points)",
		inputPath, outputPath, feature.NumPoints()))

	inputDB, err := sqlite.OpenConn(inputPath, sqlite.SQLITE_OPEN_READONLY)
	if err != nil {
		return err
	}
	defer func() {
		if inputDB != nil {
			_ = inputDB.Close()
		}
	}()

	db, err := inputDB.BackupToDB("", outputPath)
	if err != nil {
		return err
	}
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()

	err = inputDB.Close()
	inputDB = nil
	if err != nil {
		return err
	}
	slog.Info("Copied input db")

	if err := sqlitex.ExecTransient(db, "CREATE TABLE __gtfs2sqlite_stops_inside (stop_id TEXT)", sqlitexNoop); err != nil {
		return err
	}

	stopsInsideCount := 0
	totalStopCount := 0
	err = sqlitex.Exec(db, "SELECT stop_id, stop_lon, stop_lat FROM stops", func(stmt *sqlite.Stmt) error {
		stopID := stmt.GetText("stop_id")
		totalStopCount++

		lng, err := strconv.ParseFloat(stmt.GetText("stop_lon"), 64)
		if err != nil {
			slog.Error("Failed to parse stop_lon", "stop_id", stopID)
		}
		lat, err := strconv.ParseFloat(stmt.GetText("stop_lat"), 64)
		if err != nil {
			slog.Error("Failed to parse stop_lat", "stop_id", stopID)
		}
		point := geojson.NewPoint(geometry.Point{X: lng, Y: lat})

		if feature.Contains(point) {
			stopsInsideCount++
			return sqlitex.Exec(db, "INSERT INTO __gtfs2sqlite_stops_inside (stop_id) VALUES (?)", sqlitexNoop, stopID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("%d of %d stops are inside", stopsInsideCount, totalStopCount))

	script := `
DELETE FROM trips
	WHERE trip_id NOT IN (SELECT DISTINCT trip_id FROM stop_times WHERE stop_id IN __gtfs2sqlite_stops_inside);

DELETE FROM stop_times WHERE trip_id NOT IN (SELECT DISTINCT trip_id FROM trips);

DELETE FROM stops WHERE stop_id NOT IN
	(SELECT DISTINCT stop_id FROM stop_times
	 UNION SELECT DISTINCT parent_station FROM stops WHERE parent_station IS NOT NULL);

DELETE FROM stop_areas WHERE stop_id NOT IN (SELECT DISTINCT stop_id FROM stops);
DELETE FROM areas WHERE area_id NOT IN (SELECT DISTINCT area_id FROM stop_areas);

DELETE FROM routes WHERE route_id NOT IN (SELECT DISTINCT route_id FROM trips);

DELETE FROM agency WHERE agency_id NOT IN (SELECT DISTINCT agency_id FROM routes);

DELETE FROM calendar WHERE service_id NOT IN (SELECT DISTINCT service_id FROM trips);
DELETE FROM calendar_dates WHERE service_id NOT IN (SELECT DISTINCT service_id FROM trips);

DELETE FROM transfers WHERE
  (from_stop_id IS NOT NULL AND from_stop_id NOT IN (SELECT DISTINCT stop_id FROM stops)) OR
  (to_stop_id IS NOT NULL AND to_stop_id NOT IN (SELECT DISTINCT stop_id FROM stops)) OR
  (from_route_id IS NOT NULL AND from_route_id NOT IN (SELECT DISTINCT route_id FROM routes)) OR
  (to_route_id IS NOT NULL AND to_route_id NOT IN (SELECT DISTINCT route_id FROM routes)) OR
  (from_trip_id IS NOT NULL AND from_trip_id NOT IN (SELECT DISTINCT trip_id FROM trips)) OR
  (to_trip_id IS NOT NULL AND to_trip_id NOT IN (SELECT DISTINCT trip_id FROM trips));

DELETE FROM pathways WHERE
  (from_stop_id IS NOT NULL AND from_stop_id NOT IN (SELECT DISTINCT stop_id FROM stops)) OR
  (to_stop_id IS NOT NULL AND to_stop_id NOT IN (SELECT DISTINCT stop_id FROM stops));

DELETE FROM location_group_stops WHERE stop_id NOT IN (SELECT DISTINCT stop_id FROM stops);

DELETE FROM frequencies WHERE trip_id NOT IN (SELECT DISTINCT trip_id FROM trips);

DELETE FROM attributions WHERE
  (trip_id IS NOT NULL AND trip_id NOT IN (SELECT DISTINCT trip_id FROM trips)) OR
  (route_id IS NOT NULL AND route_id NOT IN (SELECT DISTINCT route_id FROM routes)) OR
  (agency_id IS NOT NULL AND agency_id NOT IN (SELECT agency_id FROM agency));

DELETE FROM fare_rules WHERE route_id IS NOT NULL AND route_id NOT IN (SELECT DISTINCT route_id FROM routes);
DELETE FROM fare_attributes WHERE
  (fare_id NOT IN (SELECT DISTINCT fare_id FROM fare_rules)) OR
  (agency_id IS NOT NULL AND agency_id NOT IN (SELECT agency_id FROM agency));

DELETE FROM fare_leg_rules WHERE
  (from_area_id IS NOT NULL AND from_area_id NOT IN (SELECT DISTINCT area_id FROM areas)) OR
  (to_area_id IS NOT NULL AND to_area_id NOT IN (SELECT DISTINCT area_id FROM areas));

DELETE FROM route_networks WHERE route_id NOT IN (SELECT DISTINCT route_id FROM routes);

DELETE FROM timeframe WHERE service_id NOT IN
  (SELECT DISTINCT service_id FROM calendar UNION SELECT DISTINCT service_id FROM calendar_dates);

DELETE FROM booking_rules WHERE
  prior_notice_service_id IS NOT NULL AND
  prior_notice_service_id NOT IN
    (SELECT DISTINCT service_id FROM calendar UNION SELECT DISTINCT service_id FROM calendar_dates);

DROP TABLE __gtfs2sqlite_stops_inside;
`
	if err := sqlitex.ExecScript(db, script); err != nil {
		return err
	}
	if _, err = validate(db, validateOpts{logLevel: slog.LevelError}); err != nil {
		return err
	}

	err = db.Close()
	db = nil
	if err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Wrote %s", outputPath))
	return nil
}
