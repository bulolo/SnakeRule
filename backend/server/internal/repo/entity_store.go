package repo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// syncEntityBucket persists one repository aggregate into its normalized tables.
// Each aggregate is written atomically while indexed columns support lookups.
func syncEntityBucket(tx *sql.Tx, bucket, payload string) error {
	statements := map[string][]string{
		"organizations.json": {
			`DELETE FROM organizations`,
			`INSERT INTO organizations(id,slug,name,description,document) SELECT key,value->>'slug',value->>'name',COALESCE(value->>'description',''),value FROM jsonb_each($1::jsonb)`,
		},
		"users.json": {
			`DELETE FROM organization_members`, `DELETE FROM project_members`, `DELETE FROM users`,
			`INSERT INTO users(id,email,name,role,document) SELECT key,value->>'email',value->>'name',value->>'role',value FROM jsonb_each($1::jsonb)`,
			`INSERT INTO organization_members(organization_id,user_id,role) SELECT membership.key, users.id, COALESCE(users.document->'organization_roles'->>membership.key,users.role) FROM users CROSS JOIN LATERAL jsonb_array_elements_text(COALESCE(users.document->'organization_ids','[]'::jsonb)) membership(key)`,
			`INSERT INTO project_members(organization_id,project_id,user_id,role) SELECT projects.organization_id, membership.key, users.id, membership.value FROM users CROSS JOIN LATERAL jsonb_each_text(COALESCE(users.document->'project_roles','{}'::jsonb)) membership JOIN projects ON projects.id=membership.key`,
		},
		"projects.json": {
			`DELETE FROM projects`,
			`INSERT INTO projects(organization_id,id,slug,name,description,document) SELECT org.key, project.key, project.value->>'slug', project.value->>'name', COALESCE(project.value->>'description',''), project.value FROM jsonb_each($1::jsonb) org CROSS JOIN LATERAL jsonb_each(org.value) project`,
		},
		"rulesets.json": {
			`DELETE FROM rulesets`,
			`INSERT INTO rulesets(organization_id,project_id,id,name,version,enabled,document) SELECT split_part(scope.key,'/',1), split_part(scope.key,'/',2), item.key, item.value->>'name', COALESCE(item.value->>'version',''), COALESCE((item.value->>'enabled')::boolean,true), item.value FROM jsonb_each($1::jsonb) scope CROSS JOIN LATERAL jsonb_each(scope.value) item`,
		},
		"project-keys.json": {
			`DELETE FROM project_credentials`,
			`INSERT INTO project_credentials(id,organization_id,project_id,name,type,prefix,secret_hash,document) SELECT key, COALESCE(value->>'Organization','default'), COALESCE(value->>'Project',''), value->'Credential'->>'name', COALESCE(value->'Credential'->>'type','apikey'), COALESCE(value->'Credential'->>'prefix',''), COALESCE(value->>'Hash',''), value FROM jsonb_each($1::jsonb)`,
		},
		"alert-channels.json": {
			`DELETE FROM alert_channels`,
			`INSERT INTO alert_channels(id,organization_id,name,type,enabled,document) SELECT key, COALESCE(value->'channel'->>'organization_id','default'), value->'channel'->>'name', value->'channel'->>'type', COALESCE((value->'channel'->>'enabled')::boolean,true), value FROM jsonb_each($1::jsonb)`,
		},
		"notifications.json": {
			`DELETE FROM notifications`,
			`INSERT INTO notifications(id,title,message,read,created_at,document) SELECT value->>'id', COALESCE(value->>'title',''), COALESCE(value->>'message',''), COALESCE((value->>'read')::boolean,false), COALESCE((value->>'created_at')::timestamptz,NOW()), value FROM jsonb_array_elements($1::jsonb) value`,
		},
		"invocations.json": {
			`DELETE FROM invocation_logs`,
			`INSERT INTO invocation_logs(id,organization_id,project_id,ruleset_id,status_code,created_at,document) SELECT value->>'id', COALESCE(value->>'organization_id',''), COALESCE(value->>'project_id',''), COALESCE(value->>'ruleset_id',''), COALESCE((value->>'status_code')::integer,0), COALESCE((value->>'created_at')::timestamptz,NOW()), value FROM jsonb_array_elements($1::jsonb) value`,
		},
		"sessions.json": {
			`DELETE FROM sessions`,
			`INSERT INTO sessions(token,email,organization_id,project_id,expires_at,document) SELECT key, COALESCE(value->>'email',''), COALESCE(value->>'organization_id','default'), COALESCE(value->>'project_id',''), COALESCE((value->>'expires_at')::timestamptz,NOW()), value FROM jsonb_each($1::jsonb)`,
		},
		"platform-settings.json": {
			`DELETE FROM platform_settings`,
			`INSERT INTO platform_settings(id,name,tagline,logo_url,document) VALUES(1,$1::jsonb->>'name',COALESCE($1::jsonb->>'tagline',''),COALESCE($1::jsonb->>'logo_url',''),$1::jsonb)`,
		},
		"license.json":    {`DELETE FROM license_state`, `INSERT INTO license_state(id,document) VALUES(1,$1::jsonb)`},
		"jwt-signing.key": {`DELETE FROM jwt_signing_keys`, `INSERT INTO jwt_signing_keys(id,document) VALUES(1,$1::jsonb)`},
	}
	queries, ok := statements[bucket]
	if !ok {
		return nil
	}
	for _, query := range queries {
		var err error
		if strings.Contains(query, "$1") {
			_, err = tx.Exec(query, payload)
		} else {
			_, err = tx.Exec(query)
		}
		if err != nil {
			return fmt.Errorf("sync %s relational table: %w", bucket, err)
		}
	}
	return nil
}

func readEntityBucket(db *sql.DB, bucket string) ([]byte, bool, error) {
	var query string
	switch bucket {
	case "organizations.json":
		query = `SELECT id, document FROM organizations ORDER BY id`
	case "users.json":
		query = `SELECT id, document FROM users ORDER BY id`
	case "projects.json":
		query = `SELECT organization_id, id, document FROM projects ORDER BY organization_id,id`
	case "rulesets.json":
		query = `SELECT organization_id, project_id, id, document FROM rulesets ORDER BY organization_id,project_id,id`
	case "project-keys.json":
		query = `SELECT id, document FROM project_credentials ORDER BY id`
	case "alert-channels.json":
		query = `SELECT id, document FROM alert_channels ORDER BY id`
	case "notifications.json":
		query = `SELECT id, document FROM notifications ORDER BY created_at,id`
	case "invocations.json":
		query = `SELECT id, document FROM invocation_logs ORDER BY created_at,id`
	case "sessions.json":
		query = `SELECT token, document FROM sessions ORDER BY token`
	case "platform-settings.json":
		query = `SELECT document FROM platform_settings WHERE id=1`
	case "license.json":
		query = `SELECT document FROM license_state WHERE id=1`
	case "jwt-signing.key":
		query = `SELECT document FROM jwt_signing_keys WHERE id=1`
	default:
		return nil, false, nil
	}
	rows, err := db.Query(query)
	if err != nil {
		return nil, false, fmt.Errorf("read %s relational table: %w", bucket, err)
	}
	defer rows.Close()
	var records []json.RawMessage
	var object = make(map[string]any)
	for rows.Next() {
		var id, organizationID, projectID string
		var raw []byte
		switch bucket {
		case "projects.json":
			if err := rows.Scan(&organizationID, &id, &raw); err != nil {
				return nil, false, err
			}
		case "rulesets.json":
			if err := rows.Scan(&organizationID, &projectID, &id, &raw); err != nil {
				return nil, false, err
			}
		case "platform-settings.json", "license.json", "jwt-signing.key":
			if err := rows.Scan(&raw); err != nil {
				return nil, false, err
			}
		default:
			if err := rows.Scan(&id, &raw); err != nil {
				return nil, false, err
			}
		}
		if bucket == "notifications.json" || bucket == "invocations.json" {
			records = append(records, append(json.RawMessage(nil), raw...))
		} else if bucket == "projects.json" || bucket == "rulesets.json" {
			if _, ok := object[organizationID]; !ok {
				object[organizationID] = make(map[string]any)
			}
			org := object[organizationID].(map[string]any)
			if bucket == "rulesets.json" {
				if _, ok := org[projectID]; !ok {
					org[projectID] = make(map[string]any)
				}
				org[projectID].(map[string]any)[id] = json.RawMessage(raw)
			} else {
				org[id] = json.RawMessage(raw)
			}
		} else if bucket == "platform-settings.json" || bucket == "license.json" || bucket == "jwt-signing.key" {
			return raw, true, nil
		} else {
			object[id] = json.RawMessage(raw)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(records) == 0 && len(object) == 0 {
		return nil, false, nil
	}
	if bucket == "notifications.json" || bucket == "invocations.json" {
		data, err := json.Marshal(records)
		return data, err == nil, err
	}
	data, err := json.Marshal(object)
	return data, err == nil, err
}
