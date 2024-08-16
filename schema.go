package gtfs2sqlite

// NOTE: Skipped validating
//   - foreign IDs to geojson, translations, and from calendar_dates / calendar service_ids

type tableSchema struct {
	PrimaryKey []string
	Columns    map[string]columnSchema
}

type columnSchema struct {
	TypeDescription     string
	PresenceDescription string
	ForeignID           *foreignIDSchema
}

type foreignIDSchema struct {
	Table  string
	Column string
	AnyOf  []foreignIDSchema
}

var gtfsSchema = map[string]tableSchema{
	"agency": {
		PrimaryKey: []string{"agency_id"},
		Columns: map[string]columnSchema{
			"agency_id":       {TypeDescription: "Unique ID", PresenceDescription: "Conditionally Required"},
			"agency_name":     {TypeDescription: "Text", PresenceDescription: "Required"},
			"agency_url":      {TypeDescription: "URL", PresenceDescription: "Required"},
			"agency_timezone": {TypeDescription: "Timezone", PresenceDescription: "Required"},
			"agency_lang":     {TypeDescription: "Language code", PresenceDescription: "Optional"},
			"agency_phone":    {TypeDescription: "Phone number", PresenceDescription: "Optional"},
			"agency_fare_url": {TypeDescription: "URL", PresenceDescription: "Optional"},
			"agency_email":    {TypeDescription: "Email", PresenceDescription: "Optional"},
		},
	},

	"stops": {
		PrimaryKey: []string{"stop_id"},
		Columns: map[string]columnSchema{
			"stop_id":             {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"stop_code":           {TypeDescription: "Text", PresenceDescription: "Optional"},
			"stop_name":           {TypeDescription: "Text", PresenceDescription: "Conditionally Required"},
			"tts_stop_name":       {TypeDescription: "Text", PresenceDescription: "Optional"},
			"stop_desc":           {TypeDescription: "Text", PresenceDescription: "Optional"},
			"stop_lat":            {TypeDescription: "Latitude", PresenceDescription: "Conditionally Required"},
			"stop_lon":            {TypeDescription: "Longitude", PresenceDescription: "Conditionally Required"},
			"zone_id":             {TypeDescription: "ID", PresenceDescription: "Optional"},
			"stop_url":            {TypeDescription: "URL", PresenceDescription: "Optional"},
			"location_type":       {TypeDescription: "Enum", PresenceDescription: "Optional"},
			"parent_station":      {TypeDescription: "Foreign ID referencing stops.stop_id", ForeignID: &foreignIDSchema{Table: "stops", Column: "stop_id"}, PresenceDescription: "Conditionally Required"},
			"stop_timezone":       {TypeDescription: "Timezone", PresenceDescription: "Optional"},
			"wheelchair_boarding": {TypeDescription: "Enum", PresenceDescription: "Optional"},
			"level_id":            {TypeDescription: "Foreign ID referencing levels.level_id", ForeignID: &foreignIDSchema{Table: "levels", Column: "level_id"}, PresenceDescription: "Optional"},
			"platform_code":       {TypeDescription: "Text", PresenceDescription: "Optional"},
		},
	},

	"routes": {
		PrimaryKey: []string{"route_id"},
		Columns: map[string]columnSchema{
			"route_id": {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"agency_id": {
				TypeDescription:     "Foreign ID referencing agency.agency_id",
				ForeignID:           &foreignIDSchema{Table: "agency", Column: "agency_id"},
				PresenceDescription: "Conditionally Required",
			},
			"route_short_name":    {TypeDescription: "Text", PresenceDescription: "Conditionally Required"},
			"route_long_name":     {TypeDescription: "Text", PresenceDescription: "Conditionally Required"},
			"route_desc":          {TypeDescription: "Text", PresenceDescription: "Optional"},
			"route_type":          {TypeDescription: "Enum", PresenceDescription: "Required"},
			"route_url":           {TypeDescription: "URL", PresenceDescription: "Optional"},
			"route_color":         {TypeDescription: "Color", PresenceDescription: "Optional"},
			"route_text_color":    {TypeDescription: "Color", PresenceDescription: "Optional"},
			"route_sort_order":    {TypeDescription: "Non-negative integer", PresenceDescription: "Optional"},
			"continuous_pickup":   {TypeDescription: "Enum", PresenceDescription: "Conditionally Forbidden"},
			"continuous_drop_off": {TypeDescription: "Enum", PresenceDescription: "Conditionally Forbidden"},
			"network_id":          {TypeDescription: "ID", PresenceDescription: "Conditionally Forbidden"},
		},
	},

	"trips": {
		PrimaryKey: []string{"trip_id"},
		Columns: map[string]columnSchema{
			"route_id": {TypeDescription: "Foreign ID referencing routes.route_id", ForeignID: &foreignIDSchema{Table: "routes", Column: "route_id"}, PresenceDescription: "Required"},
			"service_id": {
				TypeDescription: "Foreign ID referencing calendar.service_id or calendar_dates.service_id",
				ForeignID: &foreignIDSchema{AnyOf: []foreignIDSchema{
					{Table: "calendar", Column: "service_id"},
					{Table: "calendar_dates", Column: "service_id"},
				}},
				PresenceDescription: "Required",
			},
			"trip_id":         {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"trip_headsign":   {TypeDescription: "Text", PresenceDescription: "Optional"},
			"trip_short_name": {TypeDescription: "Text", PresenceDescription: "Optional"},
			"direction_id":    {TypeDescription: "Enum", PresenceDescription: "Optional"},
			"block_id":        {TypeDescription: "ID", PresenceDescription: "Optional"},
			"shape_id": {
				TypeDescription:     "Foreign ID referencing shapes.shape_id",
				ForeignID:           &foreignIDSchema{Table: "shapes", Column: "shape_id"},
				PresenceDescription: "Conditionally Required",
			},
			"wheelchair_accessible": {TypeDescription: "Enum", PresenceDescription: "Optional"},
			"bikes_allowed":         {TypeDescription: "Enum", PresenceDescription: "Optional"},
		},
	},

	"stop_times": {
		PrimaryKey: []string{"trip_id", "stop_sequence"},
		Columns: map[string]columnSchema{
			"trip_id": {
				TypeDescription:     "Foreign ID referencing trips.trip_id",
				ForeignID:           &foreignIDSchema{Table: "trips", Column: "trip_id"},
				PresenceDescription: "Required",
			},
			"arrival_time":   {TypeDescription: "Time", PresenceDescription: "Conditionally Required"},
			"departure_time": {TypeDescription: "Time", PresenceDescription: "Conditionally Required"},
			"stop_id": {
				TypeDescription:     "Foreign ID referencing stops.stop_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "stop_id"},
				PresenceDescription: "Conditionally Required",
			},
			"location_group_id": {
				TypeDescription:     "Foreign ID referencing location_groups.location_group_id",
				ForeignID:           &foreignIDSchema{Table: "location_groups", Column: "location_group_id"},
				PresenceDescription: "Conditionally Forbidden",
			},
			"location_id":                  {TypeDescription: "Foreign ID referencing id from locations.geojson", PresenceDescription: "Conditionally Forbidden"},
			"stop_sequence":                {TypeDescription: "Non-negative integer", PresenceDescription: "Required"},
			"stop_headsign":                {TypeDescription: "Text", PresenceDescription: "Optional"},
			"start_pickup_drop_off_window": {TypeDescription: "Time", PresenceDescription: "Conditionally Required"},
			"end_pickup_drop_off_window":   {TypeDescription: "Time", PresenceDescription: "Conditionally Required"},
			"pickup_type":                  {TypeDescription: "Enum", PresenceDescription: "Conditionally Forbidden"},
			"drop_off_type":                {TypeDescription: "Enum", PresenceDescription: "Conditionally Forbidden"},
			"continuous_pickup":            {TypeDescription: "Enum", PresenceDescription: "Conditionally Forbidden"},
			"continuous_drop_off":          {TypeDescription: "Enum", PresenceDescription: "Conditionally Forbidden"},
			"shape_dist_traveled":          {TypeDescription: "Non-negative float", PresenceDescription: "Optional"},
			"timepoint":                    {TypeDescription: "Enum", PresenceDescription: "Recommended"},
			"pickup_booking_rule_id": {
				TypeDescription:     "Foreign ID referencing booking_rules.booking_rule_id",
				ForeignID:           &foreignIDSchema{Table: "booking_rules", Column: "booking_rule_id"},
				PresenceDescription: "Optional",
			},
			"drop_off_booking_rule_id": {
				TypeDescription:     "Foreign ID referencing booking_rules.booking_rule_id",
				ForeignID:           &foreignIDSchema{Table: "booking_rules", Column: "booking_rule_id"},
				PresenceDescription: "Optional",
			},
		},
	},

	"calendar": {
		PrimaryKey: []string{"service_id"},
		Columns: map[string]columnSchema{
			"service_id": {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"monday":     {TypeDescription: "Enum", PresenceDescription: "Required"},
			"tuesday":    {TypeDescription: "Enum", PresenceDescription: "Required"},
			"wednesday":  {TypeDescription: "Enum", PresenceDescription: "Required"},
			"thursday":   {TypeDescription: "Enum", PresenceDescription: "Required"},
			"friday":     {TypeDescription: "Enum", PresenceDescription: "Required"},
			"saturday":   {TypeDescription: "Enum", PresenceDescription: "Required"},
			"sunday":     {TypeDescription: "Enum", PresenceDescription: "Required"},
			"start_date": {TypeDescription: "Date", PresenceDescription: "Required"},
			"end_date":   {TypeDescription: "Date", PresenceDescription: "Required"},
		},
	},

	"calendar_dates": {
		PrimaryKey: []string{"service_id", "date"},
		Columns: map[string]columnSchema{
			"service_id": {
				TypeDescription:     "Foreign ID referencing calendar.service_id or ID",
				PresenceDescription: "Required",
			},
			"date":           {TypeDescription: "Date", PresenceDescription: "Required"},
			"exception_type": {TypeDescription: "Enum", PresenceDescription: "Required"},
		},
	},

	"fare_attributes": {
		PrimaryKey: []string{"fare_id"},
		Columns: map[string]columnSchema{
			"fare_id":        {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"price":          {TypeDescription: "Non-negative float", PresenceDescription: "Required"},
			"currency_type":  {TypeDescription: "Currency code", PresenceDescription: "Required"},
			"payment_method": {TypeDescription: "Enum", PresenceDescription: "Required"},
			"transfers":      {TypeDescription: "Enum", PresenceDescription: "Required"},
			"agency_id": {
				TypeDescription:     "Foreign ID referencing agency.agency_id",
				ForeignID:           &foreignIDSchema{Table: "agency", Column: "agency_id"},
				PresenceDescription: "Conditionally Required",
			},
			"transfer_duration": {TypeDescription: "Non-negative integer", PresenceDescription: "Optional"},
		},
	},

	"fare_rules": {
		PrimaryKey: []string{"fare_id", "route_id", "origin_id", "destination_id", "contains_id"},
		Columns: map[string]columnSchema{
			"fare_id": {
				TypeDescription:     "Foreign ID referencing fare_attributes.fare_id",
				ForeignID:           &foreignIDSchema{Table: "fare_attributes", Column: "fare_id"},
				PresenceDescription: "Required",
			},
			"route_id": {
				TypeDescription:     "Foreign ID referencing routes.route_id",
				ForeignID:           &foreignIDSchema{Table: "routes", Column: "route_id"},
				PresenceDescription: "Optional",
			},
			"origin_id": {
				TypeDescription:     "Foreign ID referencing stops.zone_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "zone_id"},
				PresenceDescription: "Optional",
			},
			"destination_id": {
				TypeDescription:     "Foreign ID referencing stops.zone_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "zone_id"},
				PresenceDescription: "Optional",
			},
			"contains_id": {
				TypeDescription:     "Foreign ID referencing stops.zone_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "zone_id"},
				PresenceDescription: "Optional",
			},
		},
	},

	"timeframe": {
		PrimaryKey: []string{"timeframe_group_id", "start_time", "end_time", "service_id"},
		Columns: map[string]columnSchema{
			"timeframe_group_id": {TypeDescription: "ID", PresenceDescription: "Required"},
			"start_time":         {TypeDescription: "Time", PresenceDescription: "Conditionally Required"},
			"end_time":           {TypeDescription: "Time", PresenceDescription: "Conditionally Required"},
			"service_id": {
				TypeDescription: "Foreign ID referencing calendar.service_id or calendar_dates.service_id",
				ForeignID: &foreignIDSchema{AnyOf: []foreignIDSchema{
					{Table: "calendar", Column: "service_id"},
					{Table: "calendar_dates", Column: "service_id"},
				}},
				PresenceDescription: "Required",
			},
		},
	},

	"fare_media": {
		PrimaryKey: []string{"fare_media"},
		Columns: map[string]columnSchema{
			"fare_media_id":   {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"fare_media_name": {TypeDescription: "Text", PresenceDescription: "Optional"},
			"fare_media_type": {TypeDescription: "Enum", PresenceDescription: "Required"},
		},
	},

	"fare_products": {
		PrimaryKey: []string{"fare_product_id", "fare_media_id"},
		Columns: map[string]columnSchema{
			"fare_product_id":   {TypeDescription: "ID", PresenceDescription: "Required"},
			"fare_product_name": {TypeDescription: "Text", PresenceDescription: "Optional"},
			"fare_media_id":     {TypeDescription: "Foreign ID referencing fare_media.fare_media_id", ForeignID: &foreignIDSchema{Table: "fare_media", Column: "fare_media_id"}, PresenceDescription: "Optional"},
			"amount":            {TypeDescription: "Currency amount", PresenceDescription: "Required"},
			"currency":          {TypeDescription: "Currency code", PresenceDescription: "Required"},
		},
	},

	"fare_leg_rules": {
		PrimaryKey: []string{"network_id", "from_area_id", "to_area_id", "from_timeframe_group_id", "to_timeframe_group_id", "fare_product_id"},
		Columns: map[string]columnSchema{
			"leg_group_id": {TypeDescription: "ID", PresenceDescription: "Optional"},
			"network_id": {
				TypeDescription: "Foreign ID referencing routes.network_id or networks.network_id",
				ForeignID: &foreignIDSchema{AnyOf: []foreignIDSchema{
					{Table: "routes", Column: "network_id"},
					{Table: "networks", Column: "network_id"},
				}},
				PresenceDescription: "Optional",
			},
			"from_area_id": {TypeDescription: "Foreign ID referencing areas.area_id", ForeignID: &foreignIDSchema{
				Table:  "areas",
				Column: "area_id",
			}, PresenceDescription: "Optional"},
			"to_area_id": {
				TypeDescription:     "Foreign ID referencing areas.area_id",
				ForeignID:           &foreignIDSchema{Table: "areas", Column: "area_id"},
				PresenceDescription: "Optional",
			},
			"from_timeframe_group_id": {
				TypeDescription:     "Foreign ID referencing timeframes.timeframe_group_id",
				ForeignID:           &foreignIDSchema{Table: "timeframe", Column: "timeframe_group_id"},
				PresenceDescription: "Optional",
			},
			"to_timeframe_group_id": {
				TypeDescription:     "Foreign ID referencing timeframes.timeframe_group_id",
				ForeignID:           &foreignIDSchema{Table: "timeframe", Column: "timeframe_group_id"},
				PresenceDescription: "Optional",
			},
			"fare_product_id": {
				TypeDescription:     "Foreign ID referencing fare_products.fare_product_id",
				ForeignID:           &foreignIDSchema{Table: "fare_products", Column: "fare_product_id"},
				PresenceDescription: "Required",
			},
			"rule_priority": {TypeDescription: "Non-negative integer", PresenceDescription: "Optional"},
		},
	},

	"fare_transfer_rules": {
		PrimaryKey: []string{"from_leg_group_id", "to_leg_group_id", "fare_product_id", "transfer_count", "duration_limit"},
		Columns: map[string]columnSchema{
			"from_leg_group_id": {
				TypeDescription:     "Foreign ID referencing fare_leg_rules.leg_group_id",
				ForeignID:           &foreignIDSchema{Table: "fare_leg_rules", Column: "leg_group_id"},
				PresenceDescription: "Optional",
			},
			"to_leg_group_id": {
				TypeDescription:     "Foreign ID referencing fare_leg_rules.leg_group_id",
				ForeignID:           &foreignIDSchema{Table: "fare_leg_rules", Column: "leg_group_id"},
				PresenceDescription: "Optional",
			},
			"transfer_count":      {TypeDescription: "Non-zero integer", PresenceDescription: "Conditionally Forbidden"},
			"duration_limit":      {TypeDescription: "Positive integer", PresenceDescription: "Optional"},
			"duration_limit_type": {TypeDescription: "Enum", PresenceDescription: "Conditionally Required"},
			"fare_transfer_type":  {TypeDescription: "Enum", PresenceDescription: "Required"},
			"fare_product_id": {
				TypeDescription:     "Foreign ID referencing fare_products.fare_product_id",
				ForeignID:           &foreignIDSchema{Table: "fare_products", Column: "fare_product_id"},
				PresenceDescription: "Optional",
			},
		},
	},

	"areas": {
		PrimaryKey: []string{"area_id"},
		Columns: map[string]columnSchema{
			"area_id":   {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"area_name": {TypeDescription: "Text", PresenceDescription: "Optional"},
		},
	},

	"stop_areas": {
		PrimaryKey: []string{"area_id", "stop_id"},
		Columns: map[string]columnSchema{
			"area_id": {
				TypeDescription:     "Foreign ID referencing areas.area_id",
				ForeignID:           &foreignIDSchema{Table: "areas", Column: "area_id"},
				PresenceDescription: "Required",
			},
			"stop_id": {
				TypeDescription:     "Foreign ID referencing stops.stop_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "stop_id"},
				PresenceDescription: "Required",
			},
		},
	},

	"networks": {
		PrimaryKey: []string{"network_id"},
		Columns: map[string]columnSchema{
			"network_id":   {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"network_name": {TypeDescription: "Text", PresenceDescription: "Optional"},
		},
	},

	"route_networks": {
		PrimaryKey: []string{"route_id"},
		Columns: map[string]columnSchema{
			"network_id": {
				TypeDescription:     "Foreign ID referencing networks.network_id",
				ForeignID:           &foreignIDSchema{Table: "networks", Column: "network_id"},
				PresenceDescription: "Required",
			},
			"route_id": {
				TypeDescription:     "Foreign ID referencing routes.route_id",
				ForeignID:           &foreignIDSchema{Table: "routes", Column: "route_id"},
				PresenceDescription: "Required",
			},
		},
	},

	"shapes": {
		PrimaryKey: []string{"shape_id", "shape_pt_sequence"},
		Columns: map[string]columnSchema{
			"shape_id":            {TypeDescription: "ID", PresenceDescription: "Required"},
			"shape_pt_lat":        {TypeDescription: "Latitude", PresenceDescription: "Required"},
			"shape_pt_lon":        {TypeDescription: "Longitude", PresenceDescription: "Required"},
			"shape_pt_sequence":   {TypeDescription: "Non-negative integer", PresenceDescription: "Required"},
			"shape_dist_traveled": {TypeDescription: "Non-negative float", PresenceDescription: "Optional"},
		},
	},

	"frequencies": {
		PrimaryKey: []string{"trip_id", "start_time"},
		Columns: map[string]columnSchema{
			"trip_id": {
				TypeDescription:     "Foreign ID referencing trips.trip_id",
				ForeignID:           &foreignIDSchema{Table: "trips", Column: "trip_id"},
				PresenceDescription: "Required",
			},
			"start_time":   {TypeDescription: "Time", PresenceDescription: "Required"},
			"end_time":     {TypeDescription: "Time", PresenceDescription: "Required"},
			"headway_secs": {TypeDescription: "Positive integer", PresenceDescription: "Required"},
			"exact_times":  {TypeDescription: "Enum", PresenceDescription: "Optional"},
		},
	},

	"transfers": {
		PrimaryKey: []string{"from_stop_id", "to_stop_id", "from_trip_id", "to_trip_id", "from_route_id", "to_route_id"},
		Columns: map[string]columnSchema{
			"from_stop_id": {
				TypeDescription:     "Foreign ID referencing stops.stop_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "stop_id"},
				PresenceDescription: "Conditionally Required",
			},
			"to_stop_id": {
				TypeDescription:     "Foreign ID referencing stops.stop_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "stop_id"},
				PresenceDescription: "Conditionally Required",
			},
			"from_route_id": {
				TypeDescription:     "Foreign ID referencing routes.route_id",
				ForeignID:           &foreignIDSchema{Table: "routes", Column: "route_id"},
				PresenceDescription: "Optional",
			},
			"to_route_id": {
				TypeDescription:     "Foreign ID referencing routes.route_id",
				ForeignID:           &foreignIDSchema{Table: "routes", Column: "route_id"},
				PresenceDescription: "Optional",
			},
			"from_trip_id": {
				TypeDescription:     "Foreign ID referencing trips.trip_id",
				ForeignID:           &foreignIDSchema{Table: "trips", Column: "trip_id"},
				PresenceDescription: "Conditionally Required",
			},
			"to_trip_id": {
				TypeDescription:     "Foreign ID referencing trips.trip_id",
				ForeignID:           &foreignIDSchema{Table: "trips", Column: "trip_id"},
				PresenceDescription: "Conditionally Required",
			},
			"transfer_type":     {TypeDescription: "Enum", PresenceDescription: "Required"},
			"min_transfer_time": {TypeDescription: "Non-negative integer", PresenceDescription: "Optional"},
		},
	},

	"pathways": {
		PrimaryKey: []string{"pathway_id"},
		Columns: map[string]columnSchema{
			"pathway_id": {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"from_stop_id": {
				TypeDescription:     "Foreign ID referencing stops.stop_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "stop_id"},
				PresenceDescription: "Required",
			},
			"to_stop_id": {
				TypeDescription:     "Foreign ID referencing stops.stop_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "stop_id"},
				PresenceDescription: "Required",
			},
			"pathway_mode":           {TypeDescription: "Enum", PresenceDescription: "Required"},
			"is_bidirectional":       {TypeDescription: "Enum", PresenceDescription: "Required"},
			"length":                 {TypeDescription: "Non-negative float", PresenceDescription: "Optional"},
			"traversal_time":         {TypeDescription: "Positive integer", PresenceDescription: "Optional"},
			"stair_count":            {TypeDescription: "Non-null integer", PresenceDescription: "Optional"},
			"max_slope":              {TypeDescription: "Float", PresenceDescription: "Optional"},
			"min_width":              {TypeDescription: "Positive float", PresenceDescription: "Optional"},
			"signposted_as":          {TypeDescription: "Text", PresenceDescription: "Optional"},
			"reversed_signposted_as": {TypeDescription: "Text", PresenceDescription: "Optional"},
		},
	},

	"levels": {
		PrimaryKey: []string{"level_id"},
		Columns: map[string]columnSchema{
			"level_id":    {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"level_index": {TypeDescription: "Float", PresenceDescription: "Required"},
			"level_name":  {TypeDescription: "Text", PresenceDescription: "Optional"},
		},
	},

	"location_groups": {
		PrimaryKey: []string{"location_group_id"},
		Columns: map[string]columnSchema{
			"location_group_id":   {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"location_group_name": {TypeDescription: "Text", PresenceDescription: "Optional"},
		},
	},

	"location_group_stops": {
		PrimaryKey: []string{"location_group_stops"},
		Columns: map[string]columnSchema{
			"location_group_id": {
				TypeDescription:     "Foreign ID referencing location_groups.location_group_id",
				ForeignID:           &foreignIDSchema{Table: "location_groups", Column: "location_group_id"},
				PresenceDescription: "Required",
			},
			"stop_id": {
				TypeDescription:     "Foreign ID referencing stops.stop_id",
				ForeignID:           &foreignIDSchema{Table: "stops", Column: "stop_id"},
				PresenceDescription: "Required",
			},
		},
	},

	"booking_rules": {
		PrimaryKey: []string{"booking_rule_id"},
		Columns: map[string]columnSchema{
			"booking_rule_id":           {TypeDescription: "Unique ID", PresenceDescription: "Required"},
			"booking_type":              {TypeDescription: "Enum", PresenceDescription: "Required"},
			"prior_notice_duration_min": {TypeDescription: "Integer", PresenceDescription: "Conditionally Required"},
			"prior_notice_duration_max": {TypeDescription: "Integer", PresenceDescription: "Conditionally Forbidden"},
			"prior_notice_last_day":     {TypeDescription: "Integer", PresenceDescription: "Conditionally Required"},
			"prior_notice_last_time":    {TypeDescription: "Time", PresenceDescription: "Conditionally Required"},
			"prior_notice_start_day":    {TypeDescription: "Integer", PresenceDescription: "Conditionally Forbidden"},
			"prior_notice_start_time":   {TypeDescription: "Time", PresenceDescription: "Conditionally Required"},
			"prior_notice_service_id": {
				TypeDescription:     "Foreign ID referencing calendar.service_id",
				ForeignID:           &foreignIDSchema{Table: "calendar", Column: "service_id"},
				PresenceDescription: "Conditionally Forbidden",
			},
			"message":          {TypeDescription: "Text", PresenceDescription: "Optional"},
			"pickup_message":   {TypeDescription: "Text", PresenceDescription: "Optional"},
			"drop_off_message": {TypeDescription: "Text", PresenceDescription: "Optional"},
			"phone_number":     {TypeDescription: "Phone number", PresenceDescription: "Optional"},
			"info_url":         {TypeDescription: "URL", PresenceDescription: "Optional"},
			"booking_url":      {TypeDescription: "URL", PresenceDescription: "Optional"},
		},
	},

	"translations": {
		PrimaryKey: []string{"table_name", "field_name", "language", "record_id", "record_sub_id", "field_value"},
		Columns: map[string]columnSchema{
			"table_name":    {TypeDescription: "Enum", PresenceDescription: "Required"},
			"field_name":    {TypeDescription: "Text", PresenceDescription: "Required"},
			"language":      {TypeDescription: "Language code", PresenceDescription: "Required"},
			"translation":   {TypeDescription: "Text or URL or Email or Phone number", PresenceDescription: "Required"},
			"record_id":     {TypeDescription: "Foreign ID", PresenceDescription: "Conditionally Required"},
			"record_sub_id": {TypeDescription: "Foreign ID", PresenceDescription: "Conditionally Required"},
			"field_value":   {TypeDescription: "Text or URL or Email or Phone number", PresenceDescription: "Conditionally Required"},
		},
	},

	"feed_info": {
		PrimaryKey: nil,
		Columns: map[string]columnSchema{
			"feed_publisher_name": {TypeDescription: "Text", PresenceDescription: "Required"},
			"feed_publisher_url":  {TypeDescription: "URL", PresenceDescription: "Required"},
			"feed_lang":           {TypeDescription: "Language code", PresenceDescription: "Required"},
			"default_lang":        {TypeDescription: "Language code", PresenceDescription: "Optional"},
			"feed_start_date":     {TypeDescription: "Date", PresenceDescription: "Recommended"},
			"feed_end_date":       {TypeDescription: "Date", PresenceDescription: "Recommended"},
			"feed_version":        {TypeDescription: "Text", PresenceDescription: "Recommended"},
			"feed_contact_email":  {TypeDescription: "Email", PresenceDescription: "Optional"},
			"feed_contact_url":    {TypeDescription: "URL", PresenceDescription: "Optional"},
		},
	},

	"attributions": {
		PrimaryKey: []string{"attribution_id"},
		Columns: map[string]columnSchema{
			"attribution_id": {TypeDescription: "Unique ID", PresenceDescription: "Optional"},
			"agency_id": {
				TypeDescription:     "Foreign ID referencing agency.agency_id",
				ForeignID:           &foreignIDSchema{Table: "agency", Column: "agency_id"},
				PresenceDescription: "Optional",
			},
			"route_id": {
				TypeDescription:     "Foreign ID referencing routes.route_id",
				ForeignID:           &foreignIDSchema{Table: "routes", Column: "route_id"},
				PresenceDescription: "Optional",
			},
			"trip_id": {
				TypeDescription:     "Foreign ID referencing trips.trip_id",
				ForeignID:           &foreignIDSchema{Table: "trips", Column: "trip_id"},
				PresenceDescription: "Optional",
			},
			"organization_name": {TypeDescription: "Text", PresenceDescription: "Required"},
			"is_producer":       {TypeDescription: "Enum", PresenceDescription: "Optional"},
			"is_operator":       {TypeDescription: "Enum", PresenceDescription: "Optional"},
			"is_authority":      {TypeDescription: "Enum", PresenceDescription: "Optional"},
			"attribution_url":   {TypeDescription: "URL", PresenceDescription: "Optional"},
			"attribution_email": {TypeDescription: "Email", PresenceDescription: "Optional"},
			"attribution_phone": {TypeDescription: "Phone number", PresenceDescription: "Optional"},
		},
	},
}
