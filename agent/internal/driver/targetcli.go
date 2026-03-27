package driver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Backstore struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Mapping struct {
	TargetIQN     string `json:"target_iqn"`
	LunID         int    `json:"lun_id"`
	BackstoreType string `json:"backstore_type,omitempty"`
	BackstoreName string `json:"backstore_name,omitempty"`
}

type ACL struct {
	TargetIQN    string `json:"target_iqn"`
	InitiatorIQN string `json:"initiator_iqn"`
}

type Portal struct {
	TargetIQN string `json:"target_iqn"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
}

type CHAPConfig struct {
	Enabled       bool
	UserID        string
	PasswordSet   bool
	MutualEnabled bool
	MutualUserID  string
	MutualPassSet bool
}

type Session struct {
	SID          string `json:"sid,omitempty"`
	TargetIQN    string `json:"target_iqn,omitempty"`
	InitiatorIQN string `json:"initiator_iqn,omitempty"`
	ClientIP     string `json:"client_ip,omitempty"`
	State        string `json:"state,omitempty"`
}

type TargetDriver interface {
	Available() bool
	ListTargets(ctx context.Context) ([]string, error)
	CreateTarget(ctx context.Context, iqn string) error
	DeleteTarget(ctx context.Context, iqn string) error
}

type BackstoreDriver interface {
	Available() bool
	ListBackstores(ctx context.Context) ([]Backstore, error)
	CreateBackstore(ctx context.Context, typ, name, path, size string) error
	DeleteBackstore(ctx context.Context, typ, name string) error
}

type MappingDriver interface {
	Available() bool
	ListMappings(ctx context.Context, targetIQN string) ([]Mapping, error)
	CreateMapping(ctx context.Context, targetIQN, backstoreType, backstoreName string, lunID *int) error
	DeleteMapping(ctx context.Context, targetIQN string, lunID int) error
}

type ACLDriver interface {
	Available() bool
	ListACLs(ctx context.Context, targetIQN string) ([]ACL, error)
	CreateACL(ctx context.Context, targetIQN, initiatorIQN string) error
	DeleteACL(ctx context.Context, targetIQN, initiatorIQN string) error
}

type PortalDriver interface {
	Available() bool
	ListPortals(ctx context.Context, targetIQN string) ([]Portal, error)
	CreatePortal(ctx context.Context, targetIQN, ip string, port int) error
	DeletePortal(ctx context.Context, targetIQN, ip string, port int) error
}

type SessionDriver interface {
	Available() bool
	ListSessions(ctx context.Context) ([]Session, error)
}

type CHAPDriver interface {
	Available() bool
	GetCHAP(ctx context.Context, targetIQN string) (CHAPConfig, error)
	SetCHAP(ctx context.Context, targetIQN string, enabled bool, userID, password string) error
}

type TargetCLI struct {
	bin  string
	home string
}

func NewTargetCLI() *TargetCLI {
	home := os.Getenv("TARGETCLI_HOME")
	if home == "" {
		home = "/var/lib/casaos/iscsi-gui/targetcli"
	}

	for _, candidate := range []string{"targetcli", "targetcli-fb"} {
		if p, err := exec.LookPath(candidate); err == nil {
			return &TargetCLI{bin: p, home: home}
		}
	}
	return &TargetCLI{home: home}
}

func (d *TargetCLI) Available() bool {
	return d.bin != ""
}

func (d *TargetCLI) ListTargets(ctx context.Context) ([]string, error) {
	if !d.Available() {
		return nil, errors.New("targetcli unavailable")
	}

	out, err := d.run(ctx, "ls", "/iscsi")
	if err != nil {
		return nil, err
	}

	// Only pick top-level target lines, avoid matching ACL initiator IQNs.
	topTargetRe := regexp.MustCompile(`(?m)^\s*o-\s+(iqn\.[0-9]{4}-[0-9]{2}\.[^\s\]]+)\s+\[TPGs:\s*[0-9]+\]`)
	seen := make(map[string]struct{})
	result := make([]string, 0)

	for _, m := range topTargetRe.FindAllStringSubmatch(out, -1) {
		if len(m) < 2 {
			continue
		}
		iqn := m[1]
		if _, ok := seen[iqn]; ok {
			continue
		}
		seen[iqn] = struct{}{}
		result = append(result, iqn)
	}

	// Fallback for variant outputs that don't include [TPGs:N] marker.
	if len(result) == 0 {
		lineRe := regexp.MustCompile(`(?m)^\s*o-\s+(iqn\.[0-9]{4}-[0-9]{2}\.[^\s\]]+)`)
		for _, m := range lineRe.FindAllStringSubmatch(out, -1) {
			if len(m) < 2 {
				continue
			}
			iqn := m[1]
			if _, ok := seen[iqn]; ok {
				continue
			}
			seen[iqn] = struct{}{}
			result = append(result, iqn)
		}
	}
	return result, nil
}

func (d *TargetCLI) CreateTarget(ctx context.Context, iqn string) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}
	_, err := d.run(ctx, "/iscsi", "create", iqn)
	return err
}

func (d *TargetCLI) DeleteTarget(ctx context.Context, iqn string) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}
	_, err := d.run(ctx, "/iscsi", "delete", iqn)
	return err
}

func (d *TargetCLI) ListBackstores(ctx context.Context) ([]Backstore, error) {
	if !d.Available() {
		return nil, errors.New("targetcli unavailable")
	}

	types := []string{"fileio", "block"}
	all := make([]Backstore, 0)
	for _, typ := range types {
		items, err := d.listBackstoresByType(ctx, typ)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	return all, nil
}

func (d *TargetCLI) CreateBackstore(ctx context.Context, typ, name, path, size string) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}

	switch typ {
	case "fileio":
		if size == "" {
			return errors.New("size is required for fileio backstore")
		}
		_, err := d.run(ctx,
			"/backstores/fileio",
			"create",
			"name="+name,
			"file_or_dev="+path,
			"size="+size,
		)
		return err
	case "block":
		_, err := d.run(ctx,
			"/backstores/block",
			"create",
			"name="+name,
			"dev="+path,
		)
		return err
	default:
		return fmt.Errorf("unsupported backstore type: %s", typ)
	}
}

func (d *TargetCLI) DeleteBackstore(ctx context.Context, typ, name string) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}
	_, err := d.run(ctx, "/backstores/"+typ, "delete", name)
	return err
}

func (d *TargetCLI) ListMappings(ctx context.Context, targetIQN string) ([]Mapping, error) {
	if !d.Available() {
		return nil, errors.New("targetcli unavailable")
	}

	out, err := d.run(ctx, "ls", "/iscsi/"+targetIQN+"/tpg1/luns")
	if err != nil {
		return nil, err
	}

	lunRe := regexp.MustCompile(`\blun([0-9]+)\b`)
	bsRe := regexp.MustCompile(`/backstores/(fileio|block)/([A-Za-z0-9._-]+)`)

	seen := map[int]struct{}{}
	result := make([]Mapping, 0)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		lunMatch := lunRe.FindStringSubmatch(line)
		if len(lunMatch) < 2 {
			continue
		}
		lunID, convErr := strconv.Atoi(lunMatch[1])
		if convErr != nil {
			continue
		}
		if _, ok := seen[lunID]; ok {
			continue
		}
		seen[lunID] = struct{}{}

		m := Mapping{TargetIQN: targetIQN, LunID: lunID}
		if bsMatch := bsRe.FindStringSubmatch(line); len(bsMatch) == 3 {
			m.BackstoreType = bsMatch[1]
			m.BackstoreName = bsMatch[2]
		}
		result = append(result, m)
	}
	return result, nil
}

func (d *TargetCLI) CreateMapping(ctx context.Context, targetIQN, backstoreType, backstoreName string, lunID *int) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}
	args := []string{
		"/iscsi/" + targetIQN + "/tpg1/luns",
		"create",
		"/backstores/" + backstoreType + "/" + backstoreName,
	}
	if lunID != nil {
		args = append(args, "lun="+strconv.Itoa(*lunID))
	}
	_, err := d.run(ctx, args...)
	return err
}

func (d *TargetCLI) DeleteMapping(ctx context.Context, targetIQN string, lunID int) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}
	_, err := d.run(ctx, "/iscsi/"+targetIQN+"/tpg1/luns", "delete", "lun"+strconv.Itoa(lunID))
	return err
}

func (d *TargetCLI) ListACLs(ctx context.Context, targetIQN string) ([]ACL, error) {
	if !d.Available() {
		return nil, errors.New("targetcli unavailable")
	}

	out, err := d.run(ctx, "ls", "/iscsi/"+targetIQN+"/tpg1/acls")
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`iqn\.[0-9]{4}-[0-9]{2}\.[^\s\]]+`)
	seen := map[string]struct{}{}
	result := make([]ACL, 0)
	for _, m := range re.FindAllString(out, -1) {
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		result = append(result, ACL{TargetIQN: targetIQN, InitiatorIQN: m})
	}
	return result, nil
}

func (d *TargetCLI) CreateACL(ctx context.Context, targetIQN, initiatorIQN string) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}
	_, err := d.run(ctx, "/iscsi/"+targetIQN+"/tpg1/acls", "create", initiatorIQN)
	return err
}

func (d *TargetCLI) DeleteACL(ctx context.Context, targetIQN, initiatorIQN string) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}
	_, err := d.run(ctx, "/iscsi/"+targetIQN+"/tpg1/acls", "delete", initiatorIQN)
	return err
}

func (d *TargetCLI) ListPortals(ctx context.Context, targetIQN string) ([]Portal, error) {
	if !d.Available() {
		return nil, errors.New("targetcli unavailable")
	}

	out, err := d.run(ctx, "ls", "/iscsi/"+targetIQN+"/tpg1/portals")
	if err != nil {
		return nil, err
	}

	portalRe := regexp.MustCompile(`([0-9a-fA-F:.]+):([0-9]{1,5})`)
	seen := map[string]struct{}{}
	result := make([]Portal, 0)

	for _, m := range portalRe.FindAllStringSubmatch(out, -1) {
		if len(m) != 3 {
			continue
		}
		port, convErr := strconv.Atoi(m[2])
		if convErr != nil {
			continue
		}
		key := m[1] + ":" + m[2]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, Portal{
			TargetIQN: targetIQN,
			IP:        m[1],
			Port:      port,
		})
	}

	return result, nil
}

func (d *TargetCLI) CreatePortal(ctx context.Context, targetIQN, ip string, port int) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}
	_, err := d.run(ctx, "/iscsi/"+targetIQN+"/tpg1/portals", "create", ip, strconv.Itoa(port))
	return err
}

func (d *TargetCLI) DeletePortal(ctx context.Context, targetIQN, ip string, port int) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}
	_, err := d.run(ctx, "/iscsi/"+targetIQN+"/tpg1/portals", "delete", ip, strconv.Itoa(port))
	return err
}

func (d *TargetCLI) GetCHAP(ctx context.Context, targetIQN string) (CHAPConfig, error) {
	if !d.Available() {
		return CHAPConfig{}, errors.New("targetcli unavailable")
	}

	base := "/iscsi/" + targetIQN + "/tpg1"
	authAttrOut, err := d.run(ctx, base, "get", "attribute", "authentication")
	if err != nil {
		return CHAPConfig{}, err
	}
	userOut, err := d.run(ctx, base, "get", "auth", "userid")
	if err != nil {
		return CHAPConfig{}, err
	}
	passOut, err := d.run(ctx, base, "get", "auth", "password")
	if err != nil {
		return CHAPConfig{}, err
	}
	mutualAttrOut, _ := d.run(ctx, base, "get", "attribute", "generate_node_acls")
	mutualUserOut, _ := d.run(ctx, base, "get", "auth", "mutual_userid")
	mutualPassOut, _ := d.run(ctx, base, "get", "auth", "mutual_password")

	enabled := parseTargetCLIFlag(authAttrOut)
	userID := parseTargetCLIValue(userOut)
	password := parseTargetCLIValue(passOut)
	mutualUser := parseTargetCLIValue(mutualUserOut)
	mutualPass := parseTargetCLIValue(mutualPassOut)

	// generate_node_acls=0 generally means ACL mode (explicit auth path in use).
	// This is only surfaced as hint for UI and can be refined later.
	mutualEnabled := !parseTargetCLIFlag(mutualAttrOut)

	return CHAPConfig{
		Enabled:       enabled,
		UserID:        userID,
		PasswordSet:   password != "" && password != "(null)",
		MutualEnabled: mutualEnabled,
		MutualUserID:  mutualUser,
		MutualPassSet: mutualPass != "" && mutualPass != "(null)",
	}, nil
}

func (d *TargetCLI) SetCHAP(ctx context.Context, targetIQN string, enabled bool, userID, password string) error {
	if !d.Available() {
		return errors.New("targetcli unavailable")
	}

	base := "/iscsi/" + targetIQN + "/tpg1"
	authValue := "0"
	if enabled {
		authValue = "1"
	}
	if _, err := d.run(ctx, base, "set", "attribute", "authentication="+authValue); err != nil {
		return err
	}

	if enabled {
		if _, err := d.run(ctx, base, "set", "auth", "userid="+userID); err != nil {
			return err
		}
		if _, err := d.run(ctx, base, "set", "auth", "password="+password); err != nil {
			return err
		}
	}
	return nil
}

func (d *TargetCLI) ListSessions(ctx context.Context) ([]Session, error) {
	if !d.Available() {
		return nil, errors.New("targetcli unavailable")
	}

	out, err := d.run(ctx, "sessions")
	if err != nil {
		return nil, err
	}
	if isNoSessionsOutput(out) {
		return []Session{}, nil
	}

	targetSet := d.listTargetsSet(ctx)

	sessions := parseSessionsOutput(out, targetSet)
	for _, args := range [][]string{
		{"sessions", "detail"},
		{"sessions", "list"},
	} {
		detailOut, detailErr := d.run(ctx, args...)
		if detailErr != nil || isNoSessionsOutput(detailOut) {
			continue
		}
		sessions = pickBetterSessions(sessions, parseSessionsOutput(detailOut, targetSet))
		if sessionsHaveTargetInfo(sessions) {
			break
		}
	}

	sessions = normalizeSessionRoles(sessions, targetSet)
	return sessions, nil
}

func parseSessionsOutput(out string, targetSet map[string]struct{}) []Session {
	iqnRe := regexp.MustCompile(`iqn\.[0-9]{4}-[0-9]{2}\.[^\s\]]+`)
	ipRe := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	sidRe := regexp.MustCompile(`\bsid[:= ]+([0-9]+)\b`)
	stateRe := regexp.MustCompile(`\bstate[:= ]+([A-Za-z0-9_-]+)\b`)

	appendIfValid := func(dst []Session, s Session) []Session {
		if s.SID == "" && s.TargetIQN == "" && s.InitiatorIQN == "" && s.ClientIP == "" && s.State == "" {
			return dst
		}
		return append(dst, s)
	}

	sessions := make([]Session, 0)
	cur := Session{}
	for _, line := range strings.Split(out, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			sessions = appendIfValid(sessions, cur)
			cur = Session{}
			continue
		}

		if m := sidRe.FindStringSubmatch(strings.ToLower(t)); len(m) == 2 {
			if cur.SID != "" || cur.TargetIQN != "" || cur.InitiatorIQN != "" || cur.ClientIP != "" || cur.State != "" {
				sessions = appendIfValid(sessions, cur)
				cur = Session{}
			}
			cur.SID = m[1]
		}
		if m := stateRe.FindStringSubmatch(strings.ToLower(t)); len(m) == 2 {
			cur.State = m[1]
		}
		if m := ipRe.FindString(t); m != "" {
			cur.ClientIP = m
		}

		iqns := iqnRe.FindAllString(t, -1)
		if len(iqns) > 0 {
			targetIQN, initiatorIQN := classifySessionIQNs(iqns, targetSet)
			if cur.TargetIQN == "" && targetIQN != "" {
				cur.TargetIQN = targetIQN
			}
			if cur.InitiatorIQN == "" && initiatorIQN != "" {
				cur.InitiatorIQN = initiatorIQN
			}
		}
	}
	sessions = appendIfValid(sessions, cur)
	return sessions
}

func classifySessionIQNs(iqns []string, targetSet map[string]struct{}) (string, string) {
	if len(iqns) == 0 {
		return "", ""
	}
	if len(iqns) == 1 {
		if _, ok := targetSet[iqns[0]]; ok {
			return iqns[0], ""
		}
		return "", iqns[0]
	}

	for i, iqn := range iqns {
		if _, ok := targetSet[iqn]; ok {
			initiator := ""
			for j, other := range iqns {
				if j != i && other != iqn {
					initiator = other
					break
				}
			}
			return iqn, initiator
		}
	}

	// Fallback for unknown formats: keep previous behavior.
	return iqns[0], iqns[1]
}

func (d *TargetCLI) listTargetsSet(ctx context.Context) map[string]struct{} {
	set := make(map[string]struct{})
	targets, err := d.ListTargets(ctx)
	if err != nil {
		return set
	}
	for _, t := range targets {
		set[t] = struct{}{}
	}
	return set
}

func isNoSessionsOutput(out string) bool {
	lower := strings.ToLower(out)
	return strings.Contains(lower, "no active sessions") || strings.Contains(lower, "no sessions")
}

func parseTargetCLIValue(out string) string {
	trimmed := strings.TrimSpace(out)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "=")
	last := strings.TrimSpace(parts[len(parts)-1])
	last = strings.Trim(last, "\"'")
	last = strings.TrimSpace(last)
	if strings.EqualFold(last, "none") {
		return ""
	}
	return last
}

func parseTargetCLIFlag(out string) bool {
	v := strings.ToLower(parseTargetCLIValue(out))
	return v == "1" || v == "true" || v == "yes" || v == "on" || v == "enabled"
}

func sessionsHaveTargetInfo(items []Session) bool {
	for _, it := range items {
		if it.TargetIQN != "" || it.InitiatorIQN != "" || it.ClientIP != "" {
			return true
		}
	}
	return false
}

func pickBetterSessions(base, candidate []Session) []Session {
	if len(candidate) == 0 {
		return base
	}
	if len(base) == 0 {
		return candidate
	}

	if sessionsMetadataScore(candidate) > sessionsMetadataScore(base) {
		return candidate
	}
	return base
}

func sessionsMetadataScore(items []Session) int {
	score := 0
	for _, it := range items {
		if it.SID != "" {
			score += 1
		}
		if it.State != "" {
			score += 1
		}
		if it.TargetIQN != "" {
			score += 2
		}
		if it.InitiatorIQN != "" {
			score += 2
		}
		if it.ClientIP != "" {
			score += 2
		}
	}
	return score
}

func normalizeSessionRoles(items []Session, targetSet map[string]struct{}) []Session {
	for i := range items {
		// Some targetcli variants print only one IQN per session line. If that IQN
		// is not a known target, it should be treated as initiator instead.
		if items[i].TargetIQN != "" {
			if _, ok := targetSet[items[i].TargetIQN]; !ok && items[i].InitiatorIQN == "" {
				items[i].InitiatorIQN = items[i].TargetIQN
				items[i].TargetIQN = ""
			}
		}

		// If initiator accidentally carries a known target IQN, swap back.
		if items[i].TargetIQN == "" && items[i].InitiatorIQN != "" {
			if _, ok := targetSet[items[i].InitiatorIQN]; ok {
				items[i].TargetIQN = items[i].InitiatorIQN
				items[i].InitiatorIQN = ""
			}
		}
	}
	return items
}

func (d *TargetCLI) listBackstoresByType(ctx context.Context, typ string) ([]Backstore, error) {
	out, err := d.run(ctx, "ls", "/backstores/"+typ)
	if err != nil {
		return nil, err
	}

	tokenRe := regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	reserved := map[string]struct{}{
		"backstores":       {},
		"alua":             {},
		"default_tg_pt_gp": {},
	}
	seen := map[string]struct{}{}
	result := make([]Backstore, 0)

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		idx := strings.Index(line, "o-")
		if idx < 0 {
			continue
		}
		tail := strings.TrimSpace(line[idx+2:])
		if tail == "" {
			continue
		}
		fields := strings.Fields(tail)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if !tokenRe.MatchString(name) {
			continue
		}
		if name == typ {
			continue
		}
		if _, skip := reserved[name]; skip {
			continue
		}
		key := typ + "/" + name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, Backstore{Name: name, Type: typ})
	}
	return result, nil
}

func (d *TargetCLI) run(ctx context.Context, args ...string) (string, error) {
	home, err := d.prepareHome()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, d.bin, args...)
	if home != "" {
		cmd.Env = append(os.Environ(),
			"TARGETCLI_HOME="+home,
			"HOME="+filepath.Dir(home),
		)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("targetcli %v failed: %w: %s", args, err, string(out))
	}
	return string(out), nil
}

func (d *TargetCLI) prepareHome() (string, error) {
	if d.home == "" {
		return "", nil
	}

	if err := os.MkdirAll(d.home, 0o755); err == nil {
		return d.home, nil
	}

	fallback := "/tmp/iscsi-gui-targetcli"
	if err := os.MkdirAll(fallback, 0o755); err == nil {
		return fallback, nil
	}

	return "", fmt.Errorf("prepare targetcli home failed: %s and fallback %s are not writable", d.home, fallback)
}
