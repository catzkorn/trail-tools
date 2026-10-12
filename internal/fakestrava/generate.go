package fakestrava

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/catzkorn/trail-tools/internal/geo"
)

// historyDays is how many days of activities the athlete has, up to today.
const historyDays = 120

// home is where the athlete lives: Interlaken, between Lake Thun and Lake
// Brienz, below the Eiger, Mönch and Jungfrau.
var home = geo.Point{Lat: 46.6863, Lng: 7.8632}

// zurich is the timezone of the athlete.
var zurich = mustLoadLocation("Europe/Zurich")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("CET", 3600)
	}
	return loc
}

// sport is a kind of activity the athlete does.
type sport struct {
	sportType string
	names     []string
	// The distance in km, and the climbing in meters per km.
	minKm, maxKm                 float64
	minClimbPerKm, maxClimbPerKm float64
	// How far from home the activity starts, in km.
	trailheadKm float64
	// Photos taken during the activity.
	minPhotos, maxPhotos int
	// The hour of the day it starts.
	hour int
}

var (
	roadRun = sport{
		sportType: "Run", minKm: 7, maxKm: 12, minClimbPerKm: 1, maxClimbPerKm: 6, hour: 7,
		names: []string{"Morning Run", "Easy run along the Aare", "Tempo on the Höhematte", "Lakeside loop", "Recovery run"},
	}
	hillRun = sport{
		sportType: "TrailRun", minKm: 9, maxKm: 15, minClimbPerKm: 50, maxClimbPerKm: 75, trailheadKm: 1.5, hour: 18,
		minPhotos: 0, maxPhotos: 1,
		names: []string{"Harder Kulm repeats", "Hills after work", "Up to Heimwehfluh", "Rugen forest loop"},
	}
	longRun = sport{
		sportType: "TrailRun", minKm: 18, maxKm: 34, minClimbPerKm: 45, maxClimbPerKm: 75, trailheadKm: 8, hour: 7,
		minPhotos: 3, maxPhotos: 7,
		names: []string{
			"Augstmatthorn ridge", "Schynige Platte to Faulhorn", "Niederhorn and the Gemmenalphorn",
			"Brienzer Rothorn", "Lauterbrunnen valley and Mürren", "First to Bachalpsee", "Morgenberghorn",
		},
	}
	hike = sport{
		sportType: "Hike", minKm: 9, maxKm: 17, minClimbPerKm: 60, maxClimbPerKm: 95, trailheadKm: 12, hour: 9,
		minPhotos: 2, maxPhotos: 5,
		names: []string{"Hike to Bachalpsee", "Sulegg with friends", "Hike above Grindelwald", "Saxeten valley hike"},
	}
	ride = sport{
		sportType: "Ride", minKm: 30, maxKm: 60, minClimbPerKm: 4, maxClimbPerKm: 12, hour: 17,
		names: []string{"Ride around Lake Thun", "Ride to Brienz and back", "Evening spin"},
	}
)

var captions = []string{
	"Above the clouds", "Eiger, Mönch and Jungfrau", "Lunch with a view", "Along the ridge", "Almost at the top",
	"Lake Brienz from above", "Cows on the trail", "The last snow", "Summit selfie", "", "",
}

var descriptions = []string{
	"Legs felt good today.",
	"Perfect weather, tired legs.\n\nThe last climb was hard but the views made up for it.",
	"Easy pace, chatting the whole way.",
	"Started in the fog, came out above the clouds.",
	"",
	"",
}

// activity is a generated activity.
type activity struct {
	id          int64
	name        string
	sportType   string
	description string
	start       time.Time
	// Manual activities have no streams or map.
	manual bool

	distance           float64
	movingTime         int32
	elapsedTime        int32
	gain               float64
	avgHR, maxHR       float64
	avgCadence         float64
	avgSpeed, maxSpeed float64
	sufferScore        float64
	polyline           string
	streams            *streams
	photos             []*photo
	trailhead          geo.Point
}

// streams are the recorded data of an activity.
type streams struct {
	time     []int32
	distance []float64
	latlng   [][2]float64
	altitude []float64
	hr       []float64
	cadence  []float64
	velocity []float64
	grade    []float64
}

// photo is a photo taken during an activity.
type photo struct {
	id      string
	caption string
	taken   time.Time
	// Nil if the photo has no location.
	location *geo.Point
	// Seeds the generated image.
	seed uint64
}

// generate returns the athlete's activities from historyDays ago until now,
// newest first. Activities are the same for the same day, so their IDs stay
// stable between syncs.
func generate(now time.Time) []*activity {
	today := now.In(zurich)
	var result []*activity
	for back := range historyDays {
		day := time.Date(today.Year(), today.Month(), today.Day()-back, 0, 0, 0, 0, zurich)
		dayNumber := day.Unix() / 86400
		r := rand.New(rand.NewPCG(uint64(dayNumber), 42))
		for i, s := range plan(day, r) {
			start := day.Add(time.Duration(s.hour)*time.Hour + time.Duration(r.IntN(50))*time.Minute)
			if start.After(now) {
				continue
			}
			id := dayNumber*10 + int64(i) + 1
			if s.sportType == "Run" && day.Weekday() == time.Thursday && day.YearDay()%4 == 0 {
				result = append(result, manualRun(id, start, r))
				continue
			}
			result = append(result, newActivity(id, s, start, r))
		}
	}
	return result
}

// plan returns the activities of a day: easy runs and hill sessions during
// the week, a long trail run on Saturday, and a hike or easy run on Sunday.
func plan(day time.Time, r *rand.Rand) []sport {
	skip := r.Float64() < 0.15
	switch day.Weekday() {
	case time.Monday:
		return nil
	case time.Tuesday, time.Thursday:
		if skip {
			return nil
		}
		return []sport{roadRun}
	case time.Wednesday:
		if skip {
			return nil
		}
		return []sport{hillRun}
	case time.Friday:
		if r.Float64() < 0.35 {
			return []sport{ride}
		}
		return nil
	case time.Saturday:
		return []sport{longRun}
	case time.Sunday:
		if r.Float64() < 0.5 {
			return []sport{hike}
		}
		if skip {
			return nil
		}
		return []sport{roadRun}
	}
	return nil
}

// manualRun returns an activity entered by hand, without a track.
func manualRun(id int64, start time.Time, r *rand.Rand) *activity {
	distance := 8000 + float64(r.IntN(4000))
	moving := int32(distance / 3.1)
	return &activity{
		id:          id,
		name:        "Treadmill run",
		sportType:   "Run",
		description: "Raining all day, so the treadmill it was.",
		start:       start,
		manual:      true,
		distance:    distance,
		movingTime:  moving,
		elapsedTime: moving,
		avgSpeed:    distance / float64(moving),
		maxSpeed:    3.6,
	}
}

// newActivity generates an activity of a sport with its track.
func newActivity(id int64, s sport, start time.Time, r *rand.Rand) *activity {
	a := &activity{
		id:          id,
		name:        s.names[r.IntN(len(s.names))],
		sportType:   s.sportType,
		description: descriptions[r.IntN(len(descriptions))],
		start:       start,
	}
	targetMeters := (s.minKm + r.Float64()*(s.maxKm-s.minKm)) * 1000
	climb := (s.minClimbPerKm + r.Float64()*(s.maxClimbPerKm-s.minClimbPerKm)) * targetMeters / 1000

	// The trailhead is some way from home.
	bearing := r.Float64() * 2 * math.Pi
	a.trailhead = offset(home, s.trailheadKm*1000*math.Cos(bearing), s.trailheadKm*1000*math.Sin(bearing))
	points := loop(a.trailhead, targetMeters, r)
	dists := geo.CumulativeDistances(points)
	total := dists[len(dists)-1]

	// The elevation rises to one or two high points and comes back down.
	base := 570 + s.trailheadKm*40*r.Float64()
	profile := elevationProfile(climb, s.sportType, r)
	st := &streams{}
	var t, pausedSeconds float64
	photoAt := photoFractions(s, r)
	var maxSpeed, hrSum float64
	// The heart rate follows the effort with a lag.
	hr := 0.0
	for i, p := range points {
		f := dists[i] / total
		ele := base + profile(f) + 4*math.Sin(dists[i]/250)
		grade := 0.0
		if i > 0 {
			step := dists[i] - dists[i-1]
			prevEle := st.altitude[i-1]
			if step > 0 {
				grade = (ele - prevEle) / step
			}
			v := speed(s.sportType, grade)
			t += step / v
			maxSpeed = math.Max(maxSpeed, v)
			// Stop for photos.
			for len(photoAt) > 0 && f >= photoAt[0] {
				photoAt = photoAt[1:]
				pause := float64(30 + r.IntN(120))
				a.photos = append(a.photos, &photo{
					taken: start.Add(time.Duration(t) * time.Second),
					seed:  r.Uint64(),
				})
				if r.Float64() < 0.8 {
					loc := p
					a.photos[len(a.photos)-1].location = &loc
				}
				t += pause
				pausedSeconds += pause
			}
		}
		target := heartRate(s.sportType, grade, f, r)
		if i == 0 {
			hr = target - 25
		}
		hr += (target - hr) * 0.04
		hrSum += hr
		st.time = append(st.time, int32(t))
		st.distance = append(st.distance, math.Round(dists[i]*10)/10)
		st.latlng = append(st.latlng, [2]float64{round(p.Lat, 6), round(p.Lng, 6)})
		st.altitude = append(st.altitude, math.Round(ele*10)/10)
		st.hr = append(st.hr, math.Round(hr))
		st.cadence = append(st.cadence, cadence(s.sportType, r))
		st.velocity = append(st.velocity, math.Round(speed(s.sportType, grade)*100)/100)
		st.grade = append(st.grade, math.Round(grade*1000)/10)
	}
	for i, p := range a.photos {
		p.id = fmt.Sprintf("%d-%d", id, i+1)
		p.caption = captions[r.IntN(len(captions))]
	}
	a.streams = st
	a.distance = math.Round(total*10) / 10
	a.elapsedTime = int32(t)
	a.movingTime = int32(t - pausedSeconds)
	a.avgSpeed = total / float64(a.movingTime)
	a.maxSpeed = maxSpeed
	a.gain, _ = geo.ElevationGainLoss(st.altitude)
	a.gain = math.Round(a.gain)
	a.avgHR = math.Round(hrSum / float64(len(points)))
	a.maxHR = maxOf(st.hr)
	a.avgCadence = cadence(s.sportType, r)
	a.sufferScore = math.Round(float64(a.movingTime) / 60 * (a.avgHR - 100) / 40)
	idxs := geo.Downsample(len(points), 300)
	summary := make([]geo.Point, len(idxs))
	for i, idx := range idxs {
		summary[i] = points[idx]
	}
	a.polyline = geo.EncodePolyline(summary)
	return a
}

// loop returns the points of a loop of about the length in meters starting
// and ending at start, about 10 meters apart.
func loop(start geo.Point, meters float64, r *rand.Rand) []geo.Point {
	phase1, phase2 := r.Float64()*2*math.Pi, r.Float64()*2*math.Pi
	heading := r.Float64() * 2 * math.Pi
	wobble1, wobble2 := 0.15+0.2*r.Float64(), 0.05+0.1*r.Float64()
	shape := func(radius float64, n int) []geo.Point {
		points := make([]geo.Point, n+1)
		for i := range n + 1 {
			theta := 2 * math.Pi * float64(i) / float64(n)
			rad := radius * (1 + wobble1*math.Sin(3*theta+phase1) + wobble2*math.Sin(7*theta+phase2))
			// The loop passes through start at theta 0.
			rad0 := radius * (1 + wobble1*math.Sin(phase1) + wobble2*math.Sin(phase2))
			x := rad*math.Cos(theta) - rad0
			y := rad * math.Sin(theta)
			east := x*math.Cos(heading) - y*math.Sin(heading)
			north := x*math.Sin(heading) + y*math.Cos(heading)
			points[i] = offset(start, east, north)
		}
		points[n] = start
		return points
	}
	n := max(int(meters/10), 50)
	radius := meters / (2 * math.Pi)
	dists := geo.CumulativeDistances(shape(radius, n))
	return shape(radius*meters/dists[len(dists)-1], n)
}

// elevationProfile returns the elevation above the start at a fraction of
// the way, with high points adding up to about the climbing in meters.
func elevationProfile(climb float64, sportType string, r *rand.Rand) func(float64) float64 {
	if sportType == "Run" || sportType == "Ride" {
		amplitude := climb / 4
		phase := r.Float64() * 2 * math.Pi
		return func(f float64) float64 {
			return amplitude * (math.Sin(2*math.Pi*f*2+phase) - math.Sin(phase))
		}
	}
	type peak struct{ at, width, height float64 }
	peaks := []peak{{at: 0.35 + 0.2*r.Float64(), width: 0.18 + 0.1*r.Float64(), height: 1}}
	if r.Float64() < 0.6 {
		peaks = append(peaks, peak{at: 0.15 + 0.7*r.Float64(), width: 0.08 + 0.06*r.Float64(), height: 0.3 + 0.4*r.Float64()})
	}
	raw := func(f float64) float64 {
		sum := 0.0
		for _, p := range peaks {
			d := (f - p.at) / p.width
			sum += p.height * math.Exp(-d*d)
		}
		// Start and end at the trailhead.
		return sum * math.Pow(math.Sin(math.Pi*f), 0.6)
	}
	// Scale so that the climbing is about right.
	gain, previous := 0.0, raw(0)
	for i := 1; i <= 1000; i++ {
		value := raw(float64(i) / 1000)
		gain += math.Max(value-previous, 0)
		previous = value
	}
	scale := climb / math.Max(gain, 1e-6)
	return func(f float64) float64 { return scale * raw(f) }
}

// speed returns the speed in m/s at a grade.
func speed(sportType string, grade float64) float64 {
	switch sportType {
	case "Run":
		return math.Max(3.2*(1-4*grade), 2.2)
	case "Ride":
		return math.Min(math.Max(8*(1-8*grade), 2.5), 14)
	case "Hike":
		if grade > 0.03 {
			// About 450 vertical meters an hour.
			return math.Min(1.3, 0.125/grade)
		}
		return 1.25
	default:
		if grade > 0.05 {
			// About 800 vertical meters an hour.
			return math.Min(2.8, 0.22/grade)
		}
		if grade < -0.15 {
			return 2.4
		}
		return 2.9
	}
}

// heartRate returns the heart rate the effort at a grade leads to, a
// fraction of the way in.
func heartRate(sportType string, grade, f float64, r *rand.Rand) float64 {
	base := 140.0
	switch sportType {
	case "Hike":
		base = 115
	case "Ride":
		base = 128
	}
	// Steep climbs are walked, so the effort levels off.
	hr := base + 140*math.Min(math.Max(grade, -0.06), 0.15) + 10*f + 1.5*(r.Float64()-0.5)
	return math.Min(math.Max(hr, 95), 186)
}

// cadence returns a cadence, in steps a minute per leg for foot sports.
func cadence(sportType string, r *rand.Rand) float64 {
	switch sportType {
	case "Hike":
		return math.Round(52 + 4*r.Float64())
	case "Ride":
		return math.Round(82 + 8*r.Float64())
	}
	return math.Round(83 + 5*r.Float64())
}

// photoFractions returns where along the way photos are taken, in order.
func photoFractions(s sport, r *rand.Rand) []float64 {
	n := s.minPhotos
	if s.maxPhotos > s.minPhotos {
		n += r.IntN(s.maxPhotos - s.minPhotos + 1)
	}
	fractions := make([]float64, n)
	for i := range fractions {
		fractions[i] = 0.05 + 0.9*(float64(i)+r.Float64())/float64(max(n, 1))
	}
	// Two photos close together, as people do at the top.
	if n >= 4 {
		fractions[n/2] = fractions[n/2-1] + 0.004
	}
	return fractions
}

// offset returns the point east and north meters from p.
func offset(p geo.Point, east, north float64) geo.Point {
	return geo.Point{
		Lat: p.Lat + north/111_320,
		Lng: p.Lng + east/(111_320*math.Cos(p.Lat*math.Pi/180)),
	}
}

func round(v float64, decimals int) float64 {
	scale := math.Pow(10, float64(decimals))
	return math.Round(v*scale) / scale
}

func maxOf(values []float64) float64 {
	m := 0.0
	for _, v := range values {
		m = math.Max(m, v)
	}
	return m
}
