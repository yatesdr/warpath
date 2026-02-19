package www

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handlers) handleConfig(w http.ResponseWriter, r *http.Request) {
	cfg := h.engine.AppConfig()
	data := map[string]any{
		"Page":          "config",
		"Authenticated": h.isAuthenticated(r),
		"Config":        cfg,
		"Saved":         r.URL.Query().Get("saved"),
	}
	h.render(w, "config.html", data)
}

func (h *Handlers) handleConfigSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	section := r.FormValue("section")
	cfg := h.engine.AppConfig()

	cfg.Lock()
	switch section {
	case "general":
		// Fleet fields are part of the general section
		if v := r.FormValue("fleet_base_url"); v != "" || r.Form.Has("fleet_base_url") {
			cfg.RDS.BaseURL = v
			if d, err := time.ParseDuration(r.FormValue("fleet_poll_interval")); err == nil {
				cfg.RDS.PollInterval = d
			}
			if d, err := time.ParseDuration(r.FormValue("fleet_timeout")); err == nil {
				cfg.RDS.Timeout = d
			}
		}
	case "services":
		// Kafka brokers: indexed fields kafka_host_N / kafka_port_N
		var brokers []string
		for i := 0; ; i++ {
			host := r.FormValue(fmt.Sprintf("kafka_host_%d", i))
			if host == "" {
				break
			}
			port := r.FormValue(fmt.Sprintf("kafka_port_%d", i))
			if port == "" {
				port = "9093"
			}
			brokers = append(brokers, host+":"+port)
		}
		cfg.Messaging.Kafka.Brokers = brokers
		cfg.Messaging.Kafka.GroupID = r.FormValue("group_id")
		cfg.Messaging.OrdersTopic = r.FormValue("orders_topic")
		cfg.Messaging.DispatchTopic = r.FormValue("dispatch_topic")
		// Redis / ValKey
		cfg.Redis.Address = r.FormValue("redis_address")
		cfg.Redis.Password = r.FormValue("redis_password")
		if d, err := strconv.Atoi(r.FormValue("redis_db")); err == nil {
			cfg.Redis.DB = d
		}
	case "fleet":
		cfg.RDS.BaseURL = r.FormValue("fleet_base_url")
		if d, err := time.ParseDuration(r.FormValue("fleet_poll_interval")); err == nil {
			cfg.RDS.PollInterval = d
		}
		if d, err := time.ParseDuration(r.FormValue("fleet_timeout")); err == nil {
			cfg.RDS.Timeout = d
		}
	case "messaging":
		brokers := r.FormValue("kafka_brokers")
		if brokers != "" {
			cfg.Messaging.Kafka.Brokers = splitTrim(brokers, ",")
		} else {
			cfg.Messaging.Kafka.Brokers = []string{}
		}
		cfg.Messaging.Kafka.GroupID = r.FormValue("group_id")
		cfg.Messaging.OrdersTopic = r.FormValue("orders_topic")
		cfg.Messaging.DispatchTopic = r.FormValue("dispatch_topic")
	case "redis":
		cfg.Redis.Address = r.FormValue("redis_address")
		cfg.Redis.Password = r.FormValue("redis_password")
		if d, err := strconv.Atoi(r.FormValue("redis_db")); err == nil {
			cfg.Redis.DB = d
		}
	default:
		cfg.Unlock()
		http.Error(w, "unknown section", http.StatusBadRequest)
		return
	}
	cfg.Unlock()

	if err := cfg.Save(h.engine.ConfigPath()); err != nil {
		log.Printf("config: save error: %v", err)
		http.Error(w, "Failed to save: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Hot-reload the affected subsystem
	switch section {
	case "fleet":
		h.engine.ReconfigureFleet()
	case "general":
		h.engine.ReconfigureFleet()
	case "messaging":
		h.engine.ReconfigureMessaging()
	case "services":
		h.engine.ReconfigureMessaging()
	}

	log.Printf("config: %s section saved", section)
	http.Redirect(w, r, "/config?saved="+section, http.StatusSeeOther)
}

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
