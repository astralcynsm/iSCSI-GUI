package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"iscsi-gui/agent/internal/audit"
	"iscsi-gui/agent/internal/health"
	"iscsi-gui/agent/internal/service"
)

type Dependencies struct {
	AgentListen string
	Targets     *service.TargetsService
	Backstores  *service.BackstoresService
	Mappings    *service.MappingsService
	ACLs        *service.ACLsService
	Portals     *service.PortalsService
	CHAP        *service.CHAPService
	Sessions    *service.SessionsService
	Audit       *audit.Logger
}

type BasicHealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

type SystemHealthResponse struct {
	Status    string         `json:"status"`
	Service   string         `json:"service"`
	Timestamp string         `json:"timestamp"`
	Checks    []health.Check `json:"checks"`
}

type ListTargetsResponse struct {
	Status    string           `json:"status"`
	Service   string           `json:"service"`
	Timestamp string           `json:"timestamp"`
	Targets   []service.Target `json:"targets"`
}

type MutateTargetResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
	IQN       string `json:"iqn"`
	Message   string `json:"message"`
	Changed   bool   `json:"changed"`
}

type ListBackstoresResponse struct {
	Status     string              `json:"status"`
	Service    string              `json:"service"`
	Timestamp  string              `json:"timestamp"`
	Backstores []service.Backstore `json:"backstores"`
}

type MutateBackstoreResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	Message   string `json:"message"`
	Changed   bool   `json:"changed"`
}

type ListMappingsResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	TargetIQN string            `json:"target_iqn"`
	Mappings  []service.Mapping `json:"mappings"`
}

type MutateMappingResponse struct {
	Status        string `json:"status"`
	Service       string `json:"service"`
	Timestamp     string `json:"timestamp"`
	TargetIQN     string `json:"target_iqn"`
	LunID         *int   `json:"lun_id,omitempty"`
	BackstoreType string `json:"backstore_type,omitempty"`
	BackstoreName string `json:"backstore_name,omitempty"`
	Message       string `json:"message"`
	Changed       bool   `json:"changed"`
}

type ListACLsResponse struct {
	Status    string        `json:"status"`
	Service   string        `json:"service"`
	Timestamp string        `json:"timestamp"`
	TargetIQN string        `json:"target_iqn"`
	ACLs      []service.ACL `json:"acls"`
}

type MutateACLResponse struct {
	Status       string `json:"status"`
	Service      string `json:"service"`
	Timestamp    string `json:"timestamp"`
	TargetIQN    string `json:"target_iqn"`
	InitiatorIQN string `json:"initiator_iqn"`
	Message      string `json:"message"`
	Changed      bool   `json:"changed"`
}

type ListPortalsResponse struct {
	Status    string           `json:"status"`
	Service   string           `json:"service"`
	Timestamp string           `json:"timestamp"`
	TargetIQN string           `json:"target_iqn"`
	Portals   []service.Portal `json:"portals"`
}

type MutatePortalResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
	TargetIQN string `json:"target_iqn"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	Message   string `json:"message"`
	Changed   bool   `json:"changed"`
}

type GetCHAPResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	TargetIQN string            `json:"target_iqn"`
	CHAP      service.CHAPState `json:"chap"`
}

type SetCHAPResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	TargetIQN string            `json:"target_iqn"`
	Message   string            `json:"message"`
	Changed   bool              `json:"changed"`
	CHAP      service.CHAPState `json:"chap"`
}

type ListSessionsResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	Sessions  []service.Session `json:"sessions"`
}

type ListAuditLogsResponse struct {
	Status    string         `json:"status"`
	Service   string         `json:"service"`
	Timestamp string         `json:"timestamp"`
	Logs      []audit.Record `json:"logs"`
}

type createTargetRequest struct {
	IQN string `json:"iqn"`
}

type createBackstoreRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Size string `json:"size,omitempty"`
}

type createMappingRequest struct {
	TargetIQN     string `json:"target_iqn"`
	BackstoreType string `json:"backstore_type"`
	BackstoreName string `json:"backstore_name"`
	LunID         *int   `json:"lun_id,omitempty"`
}

type createACLRequest struct {
	TargetIQN    string `json:"target_iqn"`
	InitiatorIQN string `json:"initiator_iqn"`
}

type createPortalRequest struct {
	TargetIQN string `json:"target_iqn"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
}

type setCHAPRequest struct {
	TargetIQN string `json:"target_iqn"`
	Enabled   bool   `json:"enabled"`
	UserID    string `json:"userid,omitempty"`
	Password  string `json:"password,omitempty"`
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", basicHealth)
	mux.HandleFunc("/api/v1/system/health", systemHealth(deps.AgentListen))
	mux.HandleFunc("/api/v1/targets", targetsCollection(deps.Targets, deps.Audit))
	mux.HandleFunc("/api/v1/targets/", targetItem(deps.Targets, deps.Audit))
	mux.HandleFunc("/api/v1/backstores", backstoresCollection(deps.Backstores, deps.Audit))
	mux.HandleFunc("/api/v1/backstores/", backstoreItem(deps.Backstores, deps.Audit))
	mux.HandleFunc("/api/v1/mappings", mappingsCollection(deps.Mappings, deps.Audit))
	mux.HandleFunc("/api/v1/acls", aclsCollection(deps.ACLs, deps.Audit))
	mux.HandleFunc("/api/v1/portals", portalsCollection(deps.Portals, deps.Audit))
	mux.HandleFunc("/api/v1/auth/chap", chapCollection(deps.CHAP, deps.Audit))
	mux.HandleFunc("/api/v1/sessions", sessionsCollection(deps.Sessions))
	mux.HandleFunc("/api/v1/audit/logs", auditLogsCollection(deps.Audit))
	return mux
}

func basicHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	resp := BasicHealthResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

func systemHealth(agentListen string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
			return
		}

		report := health.Diagnose(agentListen)
		resp := SystemHealthResponse{
			Status:    report.Status,
			Service:   "iscsi-agent",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    report.Checks,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func targetsCollection(targets *service.TargetsService, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListTargets(w, r, targets)
		case http.MethodPost:
			handleCreateTarget(w, r, targets, auditLog)
		case http.MethodDelete:
			iqn := strings.TrimSpace(r.URL.Query().Get("iqn"))
			if iqn == "" {
				writeError(w, http.StatusBadRequest, "missing iqn", "use query parameter iqn or /api/v1/targets/{iqn}")
				return
			}
			handleDeleteTarget(w, r, targets, iqn, auditLog)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		}
	}
}

func targetItem(targets *service.TargetsService, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
			return
		}
		raw := strings.TrimPrefix(r.URL.Path, "/api/v1/targets/")
		if raw == "" {
			writeError(w, http.StatusBadRequest, "missing iqn", "path should be /api/v1/targets/{iqn}")
			return
		}
		iqn, err := url.PathUnescape(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid iqn in path", err.Error())
			return
		}
		handleDeleteTarget(w, r, targets, iqn, auditLog)
	}
}

func handleListTargets(w http.ResponseWriter, r *http.Request, targets *service.TargetsService) {
	if targets == nil {
		writeError(w, http.StatusServiceUnavailable, "targets service unavailable", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	items, err := targets.List(ctx)
	if err != nil {
		mapTargetError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ListTargetsResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Targets:   items,
	})
}

func handleCreateTarget(w http.ResponseWriter, r *http.Request, targets *service.TargetsService, auditLog *audit.Logger) {
	if targets == nil {
		writeError(w, http.StatusServiceUnavailable, "targets service unavailable", "")
		return
	}

	var req createTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	created, err := targets.Create(ctx, req.IQN)
	if err != nil {
		addAudit(r, auditLog, "create", "target", strings.TrimSpace(req.IQN), nil, "failed", err.Error())
		mapTargetError(w, err)
		return
	}

	code := http.StatusCreated
	msg := "target created"
	if !created {
		code = http.StatusOK
		msg = "target already exists"
	}

	writeJSON(w, code, MutateTargetResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		IQN:       strings.TrimSpace(req.IQN),
		Message:   msg,
		Changed:   created,
	})
	addAudit(r, auditLog, "create", "target", strings.TrimSpace(req.IQN), &created, "success", msg)
}

func handleDeleteTarget(w http.ResponseWriter, r *http.Request, targets *service.TargetsService, iqn string, auditLog *audit.Logger) {
	if targets == nil {
		writeError(w, http.StatusServiceUnavailable, "targets service unavailable", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	deleted, err := targets.Delete(ctx, iqn)
	if err != nil {
		addAudit(r, auditLog, "delete", "target", strings.TrimSpace(iqn), nil, "failed", err.Error())
		mapTargetError(w, err)
		return
	}

	msg := "target deleted"
	if !deleted {
		msg = "target not found"
	}

	writeJSON(w, http.StatusOK, MutateTargetResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		IQN:       strings.TrimSpace(iqn),
		Message:   msg,
		Changed:   deleted,
	})
	addAudit(r, auditLog, "delete", "target", strings.TrimSpace(iqn), &deleted, "success", msg)
}

func backstoresCollection(backstores *service.BackstoresService, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListBackstores(w, r, backstores)
		case http.MethodPost:
			handleCreateBackstore(w, r, backstores, auditLog)
		case http.MethodDelete:
			name := strings.TrimSpace(r.URL.Query().Get("name"))
			typ := strings.TrimSpace(r.URL.Query().Get("type"))
			if name == "" {
				writeError(w, http.StatusBadRequest, "missing backstore name", "use query parameter name or /api/v1/backstores/{name}")
				return
			}
			handleDeleteBackstore(w, r, backstores, typ, name, auditLog)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		}
	}
}

func backstoreItem(backstores *service.BackstoresService, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
			return
		}
		raw := strings.TrimPrefix(r.URL.Path, "/api/v1/backstores/")
		if raw == "" {
			writeError(w, http.StatusBadRequest, "missing backstore name", "path should be /api/v1/backstores/{name}")
			return
		}
		name, err := url.PathUnescape(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid backstore name in path", err.Error())
			return
		}
		typ := strings.TrimSpace(r.URL.Query().Get("type"))
		handleDeleteBackstore(w, r, backstores, typ, name, auditLog)
	}
}

func handleListBackstores(w http.ResponseWriter, r *http.Request, backstores *service.BackstoresService) {
	if backstores == nil {
		writeError(w, http.StatusServiceUnavailable, "backstores service unavailable", "")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	items, err := backstores.List(ctx)
	if err != nil {
		mapBackstoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ListBackstoresResponse{
		Status:     "ok",
		Service:    "iscsi-agent",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Backstores: items,
	})
}

func handleCreateBackstore(w http.ResponseWriter, r *http.Request, backstores *service.BackstoresService, auditLog *audit.Logger) {
	if backstores == nil {
		writeError(w, http.StatusServiceUnavailable, "backstores service unavailable", "")
		return
	}
	var req createBackstoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	created, err := backstores.Create(ctx, req.Type, req.Name, req.Path, req.Size)
	if err != nil {
		addAudit(r, auditLog, "create", "backstore", "", nil, "failed", err.Error())
		mapBackstoreError(w, err)
		return
	}

	code := http.StatusCreated
	msg := "backstore created"
	if !created {
		code = http.StatusOK
		msg = "backstore already exists"
	}

	writeJSON(w, code, MutateBackstoreResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Name:      strings.TrimSpace(req.Name),
		Type:      strings.ToLower(strings.TrimSpace(req.Type)),
		Message:   msg,
		Changed:   created,
	})
	addAudit(r, auditLog, "create", "backstore", "", &created, "success", msg+" "+strings.TrimSpace(req.Name))
}

func handleDeleteBackstore(w http.ResponseWriter, r *http.Request, backstores *service.BackstoresService, typ, name string, auditLog *audit.Logger) {
	if backstores == nil {
		writeError(w, http.StatusServiceUnavailable, "backstores service unavailable", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	deleted, err := backstores.Delete(ctx, typ, name)
	if err != nil {
		addAudit(r, auditLog, "delete", "backstore", "", nil, "failed", err.Error())
		mapBackstoreError(w, err)
		return
	}

	msg := "backstore deleted"
	if !deleted {
		msg = "backstore not found"
	}

	writeJSON(w, http.StatusOK, MutateBackstoreResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Name:      strings.TrimSpace(name),
		Type:      strings.ToLower(strings.TrimSpace(typ)),
		Message:   msg,
		Changed:   deleted,
	})
	addAudit(r, auditLog, "delete", "backstore", "", &deleted, "success", msg+" "+strings.TrimSpace(name))
}

func mappingsCollection(mappings *service.MappingsService, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			targetIQN := strings.TrimSpace(r.URL.Query().Get("target_iqn"))
			if targetIQN == "" {
				writeError(w, http.StatusBadRequest, "missing target_iqn", "use query parameter target_iqn")
				return
			}
			handleListMappings(w, r, mappings, targetIQN)
		case http.MethodPost:
			handleCreateMapping(w, r, mappings, auditLog)
		case http.MethodDelete:
			handleDeleteMapping(w, r, mappings, auditLog)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		}
	}
}

func handleListMappings(w http.ResponseWriter, r *http.Request, mappings *service.MappingsService, targetIQN string) {
	if mappings == nil {
		writeError(w, http.StatusServiceUnavailable, "mappings service unavailable", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	items, err := mappings.List(ctx, targetIQN)
	if err != nil {
		mapMappingError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ListMappingsResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TargetIQN: targetIQN,
		Mappings:  items,
	})
}

func handleCreateMapping(w http.ResponseWriter, r *http.Request, mappings *service.MappingsService, auditLog *audit.Logger) {
	if mappings == nil {
		writeError(w, http.StatusServiceUnavailable, "mappings service unavailable", "")
		return
	}

	var req createMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	changed, err := mappings.Create(ctx, req.TargetIQN, req.BackstoreType, req.BackstoreName, req.LunID)
	if err != nil {
		addAudit(r, auditLog, "create", "mapping", strings.TrimSpace(req.TargetIQN), nil, "failed", err.Error())
		mapMappingError(w, err)
		return
	}

	code := http.StatusCreated
	msg := "mapping created"
	if !changed {
		code = http.StatusOK
		msg = "mapping already exists"
	}

	writeJSON(w, code, MutateMappingResponse{
		Status:        "ok",
		Service:       "iscsi-agent",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		TargetIQN:     strings.TrimSpace(req.TargetIQN),
		LunID:         req.LunID,
		BackstoreType: strings.ToLower(strings.TrimSpace(req.BackstoreType)),
		BackstoreName: strings.TrimSpace(req.BackstoreName),
		Message:       msg,
		Changed:       changed,
	})
	addAudit(r, auditLog, "create", "mapping", strings.TrimSpace(req.TargetIQN), &changed, "success", msg)
}

func handleDeleteMapping(w http.ResponseWriter, r *http.Request, mappings *service.MappingsService, auditLog *audit.Logger) {
	if mappings == nil {
		writeError(w, http.StatusServiceUnavailable, "mappings service unavailable", "")
		return
	}
	targetIQN := strings.TrimSpace(r.URL.Query().Get("target_iqn"))
	if targetIQN == "" {
		writeError(w, http.StatusBadRequest, "missing target_iqn", "use query parameter target_iqn")
		return
	}
	lunStr := strings.TrimSpace(r.URL.Query().Get("lun_id"))
	if lunStr == "" {
		writeError(w, http.StatusBadRequest, "missing lun_id", "use query parameter lun_id")
		return
	}
	lunID, err := strconv.Atoi(lunStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lun_id", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	changed, err := mappings.Delete(ctx, targetIQN, lunID)
	if err != nil {
		addAudit(r, auditLog, "delete", "mapping", targetIQN, nil, "failed", err.Error())
		mapMappingError(w, err)
		return
	}

	msg := "mapping deleted"
	if !changed {
		msg = "mapping not found"
	}

	writeJSON(w, http.StatusOK, MutateMappingResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TargetIQN: targetIQN,
		LunID:     &lunID,
		Message:   msg,
		Changed:   changed,
	})
	addAudit(r, auditLog, "delete", "mapping", targetIQN, &changed, "success", msg)
}

func aclsCollection(acls *service.ACLsService, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			targetIQN := strings.TrimSpace(r.URL.Query().Get("target_iqn"))
			if targetIQN == "" {
				writeError(w, http.StatusBadRequest, "missing target_iqn", "use query parameter target_iqn")
				return
			}
			handleListACLs(w, r, acls, targetIQN)
		case http.MethodPost:
			handleCreateACL(w, r, acls, auditLog)
		case http.MethodDelete:
			handleDeleteACL(w, r, acls, auditLog)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		}
	}
}

func handleListACLs(w http.ResponseWriter, r *http.Request, acls *service.ACLsService, targetIQN string) {
	if acls == nil {
		writeError(w, http.StatusServiceUnavailable, "acls service unavailable", "")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	items, err := acls.List(ctx, targetIQN)
	if err != nil {
		mapACLError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ListACLsResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TargetIQN: targetIQN,
		ACLs:      items,
	})
}

func handleCreateACL(w http.ResponseWriter, r *http.Request, acls *service.ACLsService, auditLog *audit.Logger) {
	if acls == nil {
		writeError(w, http.StatusServiceUnavailable, "acls service unavailable", "")
		return
	}
	var req createACLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	changed, err := acls.Create(ctx, req.TargetIQN, req.InitiatorIQN)
	if err != nil {
		addAudit(r, auditLog, "create", "acl", strings.TrimSpace(req.TargetIQN), nil, "failed", err.Error())
		mapACLError(w, err)
		return
	}

	code := http.StatusCreated
	msg := "acl created"
	if !changed {
		code = http.StatusOK
		msg = "acl already exists"
	}

	writeJSON(w, code, MutateACLResponse{
		Status:       "ok",
		Service:      "iscsi-agent",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		TargetIQN:    strings.TrimSpace(req.TargetIQN),
		InitiatorIQN: strings.TrimSpace(req.InitiatorIQN),
		Message:      msg,
		Changed:      changed,
	})
	addAudit(r, auditLog, "create", "acl", strings.TrimSpace(req.TargetIQN), &changed, "success", msg)
}

func handleDeleteACL(w http.ResponseWriter, r *http.Request, acls *service.ACLsService, auditLog *audit.Logger) {
	if acls == nil {
		writeError(w, http.StatusServiceUnavailable, "acls service unavailable", "")
		return
	}
	targetIQN := strings.TrimSpace(r.URL.Query().Get("target_iqn"))
	if targetIQN == "" {
		writeError(w, http.StatusBadRequest, "missing target_iqn", "use query parameter target_iqn")
		return
	}
	initiatorIQN := strings.TrimSpace(r.URL.Query().Get("initiator_iqn"))
	if initiatorIQN == "" {
		writeError(w, http.StatusBadRequest, "missing initiator_iqn", "use query parameter initiator_iqn")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	changed, err := acls.Delete(ctx, targetIQN, initiatorIQN)
	if err != nil {
		addAudit(r, auditLog, "delete", "acl", targetIQN, nil, "failed", err.Error())
		mapACLError(w, err)
		return
	}

	msg := "acl deleted"
	if !changed {
		msg = "acl not found"
	}

	writeJSON(w, http.StatusOK, MutateACLResponse{
		Status:       "ok",
		Service:      "iscsi-agent",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		TargetIQN:    targetIQN,
		InitiatorIQN: initiatorIQN,
		Message:      msg,
		Changed:      changed,
	})
	addAudit(r, auditLog, "delete", "acl", targetIQN, &changed, "success", msg)
}

func sessionsCollection(sessions *service.SessionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
			return
		}
		handleListSessions(w, r, sessions)
	}
}

func chapCollection(chap *service.CHAPService, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetCHAP(w, r, chap)
		case http.MethodPut:
			handleSetCHAP(w, r, chap, auditLog)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		}
	}
}

func handleGetCHAP(w http.ResponseWriter, r *http.Request, chap *service.CHAPService) {
	if chap == nil {
		writeError(w, http.StatusServiceUnavailable, "chap service unavailable", "")
		return
	}
	targetIQN := strings.TrimSpace(r.URL.Query().Get("target_iqn"))
	if targetIQN == "" {
		writeError(w, http.StatusBadRequest, "missing target_iqn", "use query parameter target_iqn")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	state, err := chap.Get(ctx, targetIQN)
	if err != nil {
		mapCHAPError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, GetCHAPResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TargetIQN: targetIQN,
		CHAP:      state,
	})
}

func handleSetCHAP(w http.ResponseWriter, r *http.Request, chap *service.CHAPService, auditLog *audit.Logger) {
	if chap == nil {
		writeError(w, http.StatusServiceUnavailable, "chap service unavailable", "")
		return
	}

	var req setCHAPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	changed, state, err := chap.Set(ctx, req.TargetIQN, req.Enabled, req.UserID, req.Password)
	if err != nil {
		addAudit(r, auditLog, "update", "chap", strings.TrimSpace(req.TargetIQN), nil, "failed", err.Error())
		mapCHAPError(w, err)
		return
	}

	msg := "chap updated"
	if !changed {
		msg = "chap unchanged"
	}

	writeJSON(w, http.StatusOK, SetCHAPResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TargetIQN: strings.TrimSpace(req.TargetIQN),
		Message:   msg,
		Changed:   changed,
		CHAP:      state,
	})
	addAudit(r, auditLog, "update", "chap", strings.TrimSpace(req.TargetIQN), &changed, "success", msg)
}

func portalsCollection(portals *service.PortalsService, auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			targetIQN := strings.TrimSpace(r.URL.Query().Get("target_iqn"))
			if targetIQN == "" {
				writeError(w, http.StatusBadRequest, "missing target_iqn", "use query parameter target_iqn")
				return
			}
			handleListPortals(w, r, portals, targetIQN)
		case http.MethodPost:
			handleCreatePortal(w, r, portals, auditLog)
		case http.MethodDelete:
			handleDeletePortal(w, r, portals, auditLog)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		}
	}
}

func handleListPortals(w http.ResponseWriter, r *http.Request, portals *service.PortalsService, targetIQN string) {
	if portals == nil {
		writeError(w, http.StatusServiceUnavailable, "portals service unavailable", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	items, err := portals.List(ctx, targetIQN)
	if err != nil {
		mapPortalError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ListPortalsResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TargetIQN: targetIQN,
		Portals:   items,
	})
}

func handleCreatePortal(w http.ResponseWriter, r *http.Request, portals *service.PortalsService, auditLog *audit.Logger) {
	if portals == nil {
		writeError(w, http.StatusServiceUnavailable, "portals service unavailable", "")
		return
	}

	var req createPortalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if req.Port == 0 {
		req.Port = 3260
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	changed, err := portals.Create(ctx, req.TargetIQN, req.IP, req.Port)
	if err != nil {
		addAudit(r, auditLog, "create", "portal", strings.TrimSpace(req.TargetIQN), nil, "failed", err.Error())
		mapPortalError(w, err)
		return
	}

	code := http.StatusCreated
	msg := "portal created"
	if !changed {
		code = http.StatusOK
		msg = "portal already exists"
	}

	writeJSON(w, code, MutatePortalResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TargetIQN: strings.TrimSpace(req.TargetIQN),
		IP:        strings.TrimSpace(req.IP),
		Port:      req.Port,
		Message:   msg,
		Changed:   changed,
	})
	addAudit(r, auditLog, "create", "portal", strings.TrimSpace(req.TargetIQN), &changed, "success", msg)
}

func handleDeletePortal(w http.ResponseWriter, r *http.Request, portals *service.PortalsService, auditLog *audit.Logger) {
	if portals == nil {
		writeError(w, http.StatusServiceUnavailable, "portals service unavailable", "")
		return
	}

	targetIQN := strings.TrimSpace(r.URL.Query().Get("target_iqn"))
	if targetIQN == "" {
		writeError(w, http.StatusBadRequest, "missing target_iqn", "use query parameter target_iqn")
		return
	}
	ip := strings.TrimSpace(r.URL.Query().Get("ip"))
	if ip == "" {
		writeError(w, http.StatusBadRequest, "missing ip", "use query parameter ip")
		return
	}
	portStr := strings.TrimSpace(r.URL.Query().Get("port"))
	if portStr == "" {
		writeError(w, http.StatusBadRequest, "missing port", "use query parameter port")
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid port", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	changed, err := portals.Delete(ctx, targetIQN, ip, port)
	if err != nil {
		addAudit(r, auditLog, "delete", "portal", targetIQN, nil, "failed", err.Error())
		mapPortalError(w, err)
		return
	}

	msg := "portal deleted"
	if !changed {
		msg = "portal not found"
	}

	writeJSON(w, http.StatusOK, MutatePortalResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		TargetIQN: targetIQN,
		IP:        ip,
		Port:      port,
		Message:   msg,
		Changed:   changed,
	})
	addAudit(r, auditLog, "delete", "portal", targetIQN, &changed, "success", msg)
}

func auditLogsCollection(auditLog *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
			return
		}
		if auditLog == nil {
			writeError(w, http.StatusServiceUnavailable, "audit service unavailable", "")
			return
		}

		limit := 50
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid limit", err.Error())
				return
			}
			limit = n
		}
		targetIQN := strings.TrimSpace(r.URL.Query().Get("target_iqn"))
		action := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("action")))

		logs := auditLog.List(audit.Filter{
			Limit:     limit,
			TargetIQN: targetIQN,
			Action:    action,
		})

		writeJSON(w, http.StatusOK, ListAuditLogsResponse{
			Status:    "ok",
			Service:   "iscsi-agent",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Logs:      logs,
		})
	}
}

func handleListSessions(w http.ResponseWriter, r *http.Request, sessions *service.SessionsService) {
	if sessions == nil {
		writeError(w, http.StatusServiceUnavailable, "sessions service unavailable", "")
		return
	}
	targetIQN := strings.TrimSpace(r.URL.Query().Get("target_iqn"))
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	items, err := sessions.List(ctx, targetIQN)
	if err != nil {
		mapSessionError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ListSessionsResponse{
		Status:    "ok",
		Service:   "iscsi-agent",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Sessions:  items,
	})
}

func mapTargetError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidIQN):
		writeError(w, http.StatusBadRequest, "invalid iqn", "expected format: iqn.YYYY-MM.reversed.domain:name")
	case errors.Is(err, service.ErrDriverUnavailable):
		writeError(w, http.StatusServiceUnavailable, "target backend unavailable", "targetcli not found")
	default:
		writeError(w, http.StatusInternalServerError, "operation failed", err.Error())
	}
}

func mapBackstoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidBackstoreType):
		writeError(w, http.StatusBadRequest, "invalid backstore type", "supported: fileio, block")
	case errors.Is(err, service.ErrInvalidBackstoreName):
		writeError(w, http.StatusBadRequest, "invalid backstore name", "allowed: letters, numbers, dot, dash, underscore")
	case errors.Is(err, service.ErrInvalidBackstorePath):
		writeError(w, http.StatusBadRequest, "invalid backstore path", "expected absolute path like /dev/sdb or /DATA/vol1.img")
	case errors.Is(err, service.ErrInvalidBackstoreSize):
		writeError(w, http.StatusBadRequest, "invalid backstore size", "size is required for fileio, e.g. 10G")
	case errors.Is(err, service.ErrBackstoreDriverUnavailable):
		writeError(w, http.StatusServiceUnavailable, "backstore backend unavailable", "targetcli not found")
	default:
		writeError(w, http.StatusInternalServerError, "operation failed", err.Error())
	}
}

func mapMappingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidIQN):
		writeError(w, http.StatusBadRequest, "invalid target_iqn", "expected format: iqn.YYYY-MM.reversed.domain:name")
	case errors.Is(err, service.ErrInvalidBackstoreType):
		writeError(w, http.StatusBadRequest, "invalid backstore_type", "supported: fileio, block")
	case errors.Is(err, service.ErrInvalidBackstoreName):
		writeError(w, http.StatusBadRequest, "invalid backstore_name", "allowed: letters, numbers, dot, dash, underscore")
	case errors.Is(err, service.ErrInvalidLunID):
		writeError(w, http.StatusBadRequest, "invalid lun_id", "expected integer in range 0..65535")
	case errors.Is(err, service.ErrMappingDriverUnavailable):
		writeError(w, http.StatusServiceUnavailable, "mapping backend unavailable", "targetcli not found")
	default:
		writeError(w, http.StatusInternalServerError, "operation failed", err.Error())
	}
}

func mapACLError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidIQN):
		writeError(w, http.StatusBadRequest, "invalid iqn", "expected format: iqn.YYYY-MM.reversed.domain:name")
	case errors.Is(err, service.ErrACLDriverUnavailable):
		writeError(w, http.StatusServiceUnavailable, "acl backend unavailable", "targetcli not found")
	default:
		writeError(w, http.StatusInternalServerError, "operation failed", err.Error())
	}
}

func mapPortalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidIQN):
		writeError(w, http.StatusBadRequest, "invalid target_iqn", "expected format: iqn.YYYY-MM.reversed.domain:name")
	case errors.Is(err, service.ErrPortalTargetNotFound):
		writeError(w, http.StatusNotFound, "target not found", "create target first, then manage portals")
	case errors.Is(err, service.ErrInvalidPortalIP):
		writeError(w, http.StatusBadRequest, "invalid ip", "expected valid IPv4/IPv6 address")
	case errors.Is(err, service.ErrInvalidPortalPort):
		writeError(w, http.StatusBadRequest, "invalid port", "expected integer in range 1..65535")
	case errors.Is(err, service.ErrPortalDriverUnavailable):
		writeError(w, http.StatusServiceUnavailable, "portal backend unavailable", "targetcli not found")
	default:
		writeError(w, http.StatusInternalServerError, "operation failed", err.Error())
	}
}

func mapSessionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidIQN):
		writeError(w, http.StatusBadRequest, "invalid target_iqn", "expected format: iqn.YYYY-MM.reversed.domain:name")
	case errors.Is(err, service.ErrSessionDriverUnavailable):
		writeError(w, http.StatusServiceUnavailable, "session backend unavailable", "targetcli not found")
	default:
		writeError(w, http.StatusInternalServerError, "operation failed", err.Error())
	}
}

func mapCHAPError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidIQN):
		writeError(w, http.StatusBadRequest, "invalid target_iqn", "expected format: iqn.YYYY-MM.reversed.domain:name")
	case errors.Is(err, service.ErrCHAPTargetNotFound):
		writeError(w, http.StatusNotFound, "target not found", "create target first, then manage CHAP")
	case errors.Is(err, service.ErrInvalidCHAPUser):
		writeError(w, http.StatusBadRequest, "invalid userid", "userid is required when CHAP is enabled")
	case errors.Is(err, service.ErrInvalidCHAPPassword):
		writeError(w, http.StatusBadRequest, "invalid password", "password is required when CHAP is enabled")
	case errors.Is(err, service.ErrCHAPDriverUnavailable):
		writeError(w, http.StatusServiceUnavailable, "chap backend unavailable", "targetcli not found")
	default:
		writeError(w, http.StatusInternalServerError, "operation failed", err.Error())
	}
}

func writeError(w http.ResponseWriter, status int, msg, details string) {
	writeJSON(w, status, map[string]string{
		"error":   sanitizeSensitive(strings.TrimSpace(msg)),
		"details": sanitizeSensitive(strings.TrimSpace(details)),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func addAudit(r *http.Request, auditLog *audit.Logger, action, resource, targetIQN string, changed *bool, result, message string) {
	if auditLog == nil {
		return
	}
	requestID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
	if requestID == "" {
		requestID = strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	actor := strings.TrimSpace(r.Header.Get("X-User"))
	if actor == "" {
		actor = strings.TrimSpace(r.RemoteAddr)
	}

	auditLog.Add(audit.Record{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
		Actor:     actor,
		Action:    action,
		Resource:  resource,
		TargetIQN: strings.TrimSpace(targetIQN),
		Result:    result,
		Changed:   changed,
		Message:   sanitizeSensitive(message),
	})
}

var (
	sensitiveKVPattern   = regexp.MustCompile(`(?i)(password|passwd|pwd|userid)\s*=\s*[^,\s]+`)
	sensitiveJSONPattern = regexp.MustCompile(`(?i)"(password|passwd|pwd|userid)"\s*:\s*"[^"]*"`)
)

func sanitizeSensitive(in string) string {
	if in == "" {
		return ""
	}
	out := sensitiveKVPattern.ReplaceAllString(in, `$1=***`)
	out = sensitiveJSONPattern.ReplaceAllString(out, `"$1":"***"`)
	return out
}
