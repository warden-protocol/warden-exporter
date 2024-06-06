package collector

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql" // mysql driver
	"github.com/prometheus/client_golang/prometheus"

	"github.com/warden-protocol/warden-exporter/pkg/config"
	log "github.com/warden-protocol/warden-exporter/pkg/logger"
)

const (
	warpUsersMetricName      = "warp_users"
	warpQuestsDoneMetricName = "warp_quests_done"
)

type QuestDetail struct {
	Description string `json:"description"`
	Count       int    `json:"count"`
	Type        string `json:"type"`
}

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var warpUsers = prometheus.NewDesc(
	warpUsersMetricName,
	"Returns the total number of users in the Warp leaderboard",
	[]string{
		"chain_id",
		"status",
	},
	nil,
)

//nolint:gochecknoglobals // this is needed as it's used in multiple places
var warpQuestsDone = prometheus.NewDesc(
	warpQuestsDoneMetricName,
	"Returns the total number of done per quests",
	[]string{
		"chain_id",
		"description",
		"type",
		"status",
	},
	nil,
)

type WarpCollector struct {
	Cfg config.Config
}

func (w WarpCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- warpUsers
	ch <- warpQuestsDone
}

func (w WarpCollector) Collect(ch chan<- prometheus.Metric) {
	status := successStatus
	connStr := fmt.Sprintf(
		"%s:%s@tcp(%s:3306)/%s",
		w.Cfg.WarpDBUser,
		w.Cfg.WarpDBPass,
		w.Cfg.WarpDBHost,
		w.Cfg.WarpDB,
	)

	// Open database connection
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		log.Error(err.Error())
		status = errorStatus
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Error(err.Error())
		status = errorStatus
	}

	// User count
	userCount, err := queryWarpUsers(db)
	if err != nil {
		log.Error(err.Error())
		status = errorStatus
	}

	ch <- prometheus.MustNewConstMetric(
		warpUsers,
		prometheus.GaugeValue,
		float64(userCount),
		[]string{
			w.Cfg.ChainID,
			status,
		}...,
	)

	// Quests count
	quests, err := queryWarpQuestsDone(db)
	if err != nil {
		log.Error(err.Error())
		status = errorStatus
	}
	for _, q := range quests {
		ch <- prometheus.MustNewConstMetric(
			warpQuestsDone,
			prometheus.GaugeValue,
			float64(q.Count),
			[]string{
				w.Cfg.ChainID,
				q.Description,
				q.Type,
				status,
			}...,
		)
	}
}

func queryWarpUsers(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error querying users: %w", err)
	}
	return count, nil
}

func queryWarpQuestsDone(db *sql.DB) ([]QuestDetail, error) {
	var questDetails []QuestDetail
	rows, err := db.Query(
		`SELECT t2.description, t2.type, COUNT(*) as count
    FROM quests_history t1
    JOIN quests t2 ON t1.quest_pid = t2.pid
    GROUP BY t1.quest_pid, t2.description ORDER BY count DESC;`)
	if err != nil {
		return nil, fmt.Errorf("error querying quests: %w", err)
	}
	for rows.Next() {
		var qd QuestDetail
		var questType sql.NullString
		if err = rows.Scan(&qd.Description, &questType, &qd.Count); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		if questType.Valid {
			qd.Type = questType.String
		} else {
			qd.Type = "NULL"
		}
		questDetails = append(questDetails, qd)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return questDetails, nil
}
