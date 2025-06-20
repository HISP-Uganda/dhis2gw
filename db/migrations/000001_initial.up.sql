CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS plpython3u;
CREATE EXTENSION postgis;
CREATE EXTENSION xml2;

CREATE TABLE orgunitlevel
(
    id      SERIAL       NOT NULL PRIMARY KEY,
    uid     TEXT         NOT NULL UNIQUE,
    name    VARCHAR(230) NOT NULL UNIQUE,
    code    VARCHAR(50) UNIQUE,
    level   INTEGER      NOT NULL UNIQUE,
    created TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orgunitgroup
(
    id        SERIAL       NOT NULL PRIMARY KEY,
    uid       TEXT         NOT NULL UNIQUE,
    code      VARCHAR(50) UNIQUE,
    name      VARCHAR(230) NOT NULL UNIQUE,
    shortname VARCHAR(50)  NOT NULL DEFAULT '' UNIQUE,
    created   TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,
    updated   TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP

);
CREATE INDEX orgunitgroup_name_idx ON orgunitgroup (id);

CREATE TABLE attribute
(
    id                        SERIAL       NOT NULL PRIMARY KEY,
    uid                       TEXT         NOT NULL UNIQUE,
    code                      VARCHAR(50) UNIQUE,
    name                      VARCHAR(230) NOT NULL UNIQUE,
    shortname                 VARCHAR(50)  NOT NULL DEFAULT '',
    valuetype                 VARCHAR(50)  NOT NULL DEFAULT '',
    isunique                  BOOLEAN      NOT NULL DEFAULT FALSE,
    mandatory                 BOOLEAN      NOT NULL DEFAULT FALSE,
    organisationunitattribute BOOLEAN      NOT NULL,
    created                   TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP,
    updated                   TIMESTAMPTZ           DEFAULT CURRENT_TIMESTAMP

);

CREATE TABLE organisationunit
(
    id              BIGSERIAL NOT NULL PRIMARY KEY,
    uid             TEXT      NOT NULL UNIQUE,
    code            VARCHAR(50) UNIQUE,
    name            TEXT      NOT NULL DEFAULT '',
    shortname       TEXT      NOT NULL DEFAULT '',
    description     TEXT      NOT NULL DEFAULT '',
    parentid        BIGINT REFERENCES organisationunit (id),
    hierarchylevel  INTEGER   NOT NULL,
    path            TEXT      NOT NULL UNIQUE,
    address         TEXT      NOT NULL DEFAULT '',
    email           TEXT      NOT NULL DEFAULT '',
    url             TEXT      NOT NULL DEFAULT '',
    phonenumber     TEXT      NOT NULL DEFAULT '',
    extras          JSONB     NOT NULL DEFAULT '{}'::jsonb,
    attributevalues JSONB              DEFAULT '{}'::jsonb,
    mflid           TEXT,
    mfluid          TEXT,
    mflparent       TEXT,
    openingdate     DATE,
    deleted         BOOLEAN   NOT NULL DEFAULT FALSE,
    geometry        geometry(Geometry, 4326),
    lastsyncdate    TIMESTAMPTZ,
    created         TIMESTAMPTZ        DEFAULT CURRENT_TIMESTAMP,
    updated         TIMESTAMPTZ        DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX organisationunit_name_idx ON organisationunit (name);
CREATE INDEX organisationunit_level_idx ON organisationunit (hierarchylevel);
CREATE INDEX organisationunit_path_idx ON organisationunit (path);
CREATE INDEX organisationunit_mflid_idx ON organisationunit (mflid);
CREATE INDEX organisationunit_mflparent_idx ON organisationunit (mflparent);
CREATE INDEX organisationunit_parent_idx ON organisationunit (parentid);
CREATE INDEX organisationunit_created_idx ON organisationunit (created);
CREATE INDEX organisationunit_updated_idx ON organisationunit (updated);
CREATE INDEX organisationunit_geometry_idx ON organisationunit USING GIST (geometry);

CREATE TABLE orgunitgroupmembers
(
    organisationunitid BIGSERIAL REFERENCES organisationunit (id),
    orgunitgroupid     SERIAL REFERENCES orgunitgroup (id),
    created            TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated            TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organisationunitid, orgunitgroupid)

);

CREATE TABLE orgunitfailure
(
    id           BIGSERIAL NOT NULL PRIMARY KEY,
    uid          TEXT      NOT NULL UNIQUE,
    facility_uid TEXT      NOT NULL,
    mfluid       TEXT      NOT NULL,
    action       TEXT,
    reason       TEXT,
    object       JSONB     NOT NULL DEFAULT '{}'::jsonb,
    created      TIMESTAMPTZ        DEFAULT CURRENT_TIMESTAMP,
    updated      TIMESTAMPTZ        DEFAULT CURRENT_TIMESTAMP

);
CREATE INDEX orgunitfailure_uid_idx ON orgunitfailure(uid);
CREATE INDEX orgunitfailure_facility_uid_idx ON orgunitfailure(facility_uid);
CREATE INDEX orgunitfailure_mfluid_idx ON orgunitfailure(mfluid);
CREATE INDEX orgunitfailure_created_idx ON orgunitfailure(created);

CREATE TABLE IF NOT EXISTS user_roles
(
    id          BIGSERIAL NOT NULL PRIMARY KEY,
    name        TEXT      NOT NULL UNIQUE,
    description text        DEFAULT '',
    created     timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated     timestamptz DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_role_permissions
(
    id         bigserial   NOT NULL PRIMARY KEY,
    user_role  BIGINT      NOT NULL REFERENCES user_roles ON DELETE CASCADE ON UPDATE CASCADE,
    sys_module TEXT        NOT NULL, -- the name of the module - defined above this level
    sys_perms  VARCHAR(16) NOT NULL,
    created    timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated    timestamptz DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (sys_module, user_role)
);

CREATE TABLE IF NOT EXISTS users
(
    id                bigserial NOT NULL PRIMARY KEY,
    uid               TEXT      NOT NULL DEFAULT '',
    user_role         BIGINT    NOT NULL REFERENCES user_roles ON DELETE RESTRICT ON UPDATE CASCADE,
    username          TEXT      NOT NULL UNIQUE,
    password          TEXT      NOT NULL, -- blowfish hash of password
    onetime_password  TEXT,
    firstname         TEXT      NOT NULL,
    lastname          TEXT      NOT NULL,
    telephone         TEXT      NOT NULL DEFAULT '',
    email             TEXT,
    is_active         BOOLEAN   NOT NULL DEFAULT 't',
    is_admin_user    BOOLEAN   NOT NULL DEFAULT 'f',
    failed_attempts   TEXT               DEFAULT '0/' || to_char(NOW(), 'YYYYmmdd'),
    transaction_limit TEXT               DEFAULT '0/' || to_char(NOW(), 'YYYYmmdd'),
    last_login        timestamptz,
    created           timestamptz        DEFAULT CURRENT_TIMESTAMP,
    updated           timestamptz        DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX users_username_idx ON users (username);

CREATE TABLE IF NOT EXISTS user_apitoken
(
    id        bigserial NOT NULL PRIMARY KEY,
    user_id   BIGINT    NOT NULL REFERENCES users (id) ON DELETE CASCADE ON UPDATE CASCADE,
    token     TEXT      NOT NULL DEFAULT '',
    is_active BOOLEAN   NOT NULL DEFAULT TRUE,
    expires_at timestamptz DEFAULT NULL, -- if NULL, token does not expire
    created   timestamptz        DEFAULT CURRENT_TIMESTAMP,
    updated   timestamptz        DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE servers
(
    id                      serial PRIMARY KEY NOT NULL,
    uid                     TEXT               NOT NULL DEFAULT '',
    name                    text               NOT NULL UNIQUE,
    username                text               NOT NULL DEFAULT '',
    password                text               NOT NULL DEFAULT '',
    auth_token              text               NOT NULL DEFAULT '',
    ipaddress               text               NOT NULL DEFAULT '',
    url                     text               NOT NULL DEFAULT '',  -- endpoint
    callback_url            text               NOT NULL DEFAULT '',  -- url to call with response from endpoint
    cc_urls                 TEXT[]                      DEFAULT ARRAY []::TEXT[],
    http_method             text               NOT NULL DEFAULT 'POST',
    auth_method             text               NOT NULL DEFAULT '',
    system_type             TEXT               NOT NULL DEFAULT '',
    endpoint_type           TEXT               NOT NULL DEFAULT '',
    url_params              JSONB              NOT NULL DEFAULT '{}'::jsonb,
    allow_callbacks         BOOLEAN            NOT NULL DEFAULT 'f', --whether to make callbacks when destination app returns successfully
    allow_copies            BOOLEAN            NOT NULL DEFAULT 'f', --whether to allow copies to other servers
    use_async               BOOLEAN            NOT NULL DEFAULT 'f', -- whether to make async calls
    use_ssl                 BOOLEAN            NOT NULL DEFAULT 'f', --whether ssl is enabled for this server/app
    parse_responses         BOOLEAN            NOT NULL DEFAULT 't', --whether to parse responses from this server/app
    ssl_client_certkey_file TEXT               NOT NULL DEFAULT '',
    start_submission_period INTEGER            NOT NULL DEFAULT 0,   -- starting hour for off peak period
    end_submission_period   INTEGER            NOT NULL DEFAULT 24,  -- ending hour for off peak period
    xml_response_xpath      TEXT               NOT NULL DEFAULT '',
    json_response_xpath     TEXT               NOT NULL DEFAULT '',
    suspended               BOOLEAN            NOT NULL DEFAULT 'f', --whether the app, sever or endpoint is suspended
    created                 timestamptz                 DEFAULT current_timestamp,
    updated                 timestamptz                 DEFAULT current_timestamp
);

CREATE INDEX servers_name ON servers (name);
CREATE INDEX servers_uid ON servers (uid);
CREATE INDEX servers_created_idx ON servers (created);

CREATE TABLE server_allowed_sources
(
    id              serial PRIMARY KEY NOT NULL,
    server_id       INTEGER            NOT NULL REFERENCES servers (id),
    allowed_sources INTEGER[]          NOT NULL DEFAULT ARRAY []::INTEGER[],
    created         timestamptz                 DEFAULT current_timestamp,
    updated         timestamptz                 DEFAULT current_timestamp,
    UNIQUE (server_id)
);

CREATE TABLE requests
(
    id                  bigserial PRIMARY KEY NOT NULL,
    uid                 VARCHAR(11)           NOT NULL DEFAULT '',
    source              INTEGER REFERENCES servers (id),           -- source app/server
    destination         INTEGER REFERENCES servers (id),           -- source app/server
    depends_on          BIGINT REFERENCES requests (id),
    cc_servers          INTEGER[]             NOT NULL DEFAULT ARRAY []::INT[],
    cc_servers_status   JSONB                          DEFAULT '{}'::JSONB,
    batchid             TEXT                  NOT NULL DEFAULT '',
    body                TEXT                  NOT NULL DEFAULT '',
    response            TEXT                  NOT NULL DEFAULT '',
    body_is_query_param BOOLEAN               NOT NULL DEFAULT 'f',
    url_suffix          TEXT                           DEFAULT '', -- if present, it is added to API url
    ctype               TEXT                  NOT NULL DEFAULT '',
    status              VARCHAR(32)           NOT NULL DEFAULT 'ready' CHECK ( status IN
                                                                               ('pending', 'ready', 'inprogress',
                                                                                'failed', 'error', 'expired',
                                                                                'completed', 'canceled')),
    statuscode          text                           DEFAULT '',
    retries             INTEGER               NOT NULL DEFAULT 0,
    errors              TEXT                           DEFAULT '', -- indicative response message
    is_async            BOOLEAN               NOT NULL DEFAULT FALSE,
    async_jobid         TEXT                  NOT NULL DEFAULT '',
    async_response      TEXT                  NOT NULL DEFAULT '',
    async_status        TEXT                  NOT NULL DEFAULT '',
    submissionid        TEXT                  NOT NULL DEFAULT '', -- message_id in source app -> helpful when check for already sent submissions
    frequency_type      TEXT                  NOT NULL DEFAULT '',
    period              TEXT                  NOT NULL DEFAULT '', --whether ssl is enabled for this server/app
    week                TEXT                           DEFAULT '', -- reporting week
    month               TEXT                           DEFAULT '', -- reporting month
    year                INTEGER,                                   -- year of submission
    msisdn              TEXT                  NOT NULL DEFAULT '', -- can be report sender in source
    raw_msg             TEXT                  NOT NULL DEFAULT '', -- raw message in source system
    facility            TEXT                  NOT NULL DEFAULT '', -- facility owning report
    district            TEXT                  NOT NULL DEFAULT '', -- district
    report_type         TEXT                  NOT NULL DEFAULT '',
    object_type         TEXT                  NOT NULL DEFAULT '',
    extras              TEXT                  NOT NULL DEFAULT '',
    suspended           INT                   NOT NULL DEFAULT 0,  --whether to suspend this request 0 = false, 1 = true
    created             timestamptz                    DEFAULT current_timestamp,
    updated             timestamptz                    DEFAULT current_timestamp
);

CREATE INDEX requests_submissionid ON requests (submissionid);
CREATE INDEX requests_status ON requests (status);
CREATE INDEX requests_statuscode ON requests (statuscode);
CREATE INDEX requests_batchid ON requests (batchid);
CREATE INDEX requests_created ON requests (created);
CREATE INDEX requests_updated ON requests (updated);
CREATE INDEX requests_msisdn ON requests (msisdn);
CREATE INDEX requests_facility ON requests (facility);
CREATE INDEX requests_district ON requests (district);
CREATE INDEX requests_ctype ON requests (ctype);
CREATE INDEX requests_uid ON requests (uid);
CREATE INDEX requests_depends_on ON requests (depends_on);



CREATE TABLE blacklist
(
    id      bigserial PRIMARY KEY,
    msisdn  text        NOT NULL,
    created timestamptz NOT NULL DEFAULT current_timestamp,
    updated timestamptz          DEFAULT current_timestamp
);
CREATE INDEX blacklist_msisdn ON blacklist (msisdn);
CREATE INDEX blacklist_created ON blacklist (created);
CREATE INDEX blacklist_updated ON blacklist (updated);

CREATE TABLE audit_log
(
    id         BIGSERIAL   NOT NULL PRIMARY KEY,
    logtype    VARCHAR(32) NOT NULL DEFAULT '',
    actor      TEXT        NOT NULL,
    action     text        NOT NULL DEFAULT '',
    remote_ip  INET,
    detail     TEXT        NOT NULL,
    created_by INTEGER REFERENCES users (id), -- like actor id
    created    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX audit_log_created ON audit_log (created);
CREATE INDEX audit_log_logtype ON audit_log (logtype);
CREATE INDEX audit_log_action ON audit_log (action);

CREATE TABLE schedules
(
    id              bigserial   NOT NULL PRIMARY KEY,
    sched_type      TEXT        NOT NULL DEFAULT 'sms'
        CHECK (sched_type IN ('sms', 'contact_push', 'url', 'command', 'dhis2_async_job_check')), -- also 'push_contact'
    params          JSON        NOT NULL DEFAULT '{}'::json,
    sched_url       TEXT                 DEFAULT '',
    sched_content   TEXT,                                                                               -- body of scheduled url call
    command         TEXT                 DEFAULT '',
    command_args    TEXT,
    repeat          varchar(16) NOT NULL DEFAULT 'never' CHECK (
        repeat IN ('never', 'hourly', 'daily', 'weekly', 'monthly', 'yearly', 'interval', 'cron')),
    repeat_interval INTEGER,
    cron_expression TEXT,
    first_run_at    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,                                     -- when to push first.
    last_run_at     TIMESTAMPTZ,                                                                        -- when last ran
    next_run_at     TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status          text        NOT NULL DEFAULT 'ready' CHECK (
        status IN ('ready', 'skipped', 'expired', 'canceled','sent', 'failed', 'error', 'completed')),
    is_active       BOOLEAN     NOT NULL DEFAULT 't',
    async_job_type TEXT DEFAULT '',
    async_jobid     TEXT                 DEFAULT '',                                                    -- if it is an async job
    request_id      BIGINT REFERENCES requests (id),
    server_id       BIGINT REFERENCES servers (id),
    server_in_cc    BOOLEAN,
    created_by      INTEGER REFERENCES users (id),
    created         TIMESTAMPTZ          DEFAULT CURRENT_TIMESTAMP,
    updated         TIMESTAMPTZ          DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX schedules_created ON schedules(created);
CREATE INDEX schedules_first_run_at ON schedules(first_run_at);
CREATE INDEX schedules_last_run_at ON schedules(last_run_at);
CREATE INDEX schedules_next_run_at ON schedules(next_run_at);

-- FUNCTIONS
-- Check if source is an allowed 'source' for destination server/app dest
CREATE OR REPLACE FUNCTION is_allowed_source(source integer, dest integer) RETURNS BOOLEAN AS
$delim$
DECLARE
    t boolean;
BEGIN
    select source = ANY (allowed_sources) INTO t FROM server_allowed_sources WHERE server_id = dest;
    RETURN t;
END;
$delim$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION ou_paraent_from_path(ipath text, lvl INT) RETURNS BIGINT AS
$delim$
DECLARE
    i          BIGINT;
    parent_uid TEXT;
BEGIN
    SELECT split_part(ipath, '/', lvl) INTO parent_uid;
    IF FOUND THEN

        SELECT id INTO i FROM organisationunit WHERE uid = parent_uid;
        RETURN i;
    END IF;
    RETURN NULL;
END;
$delim$ LANGUAGE plpgsql;


CREATE OR REPLACE FUNCTION get_server_apps(xid INT) RETURNS TEXT AS
$delim$
DECLARE
    r TEXT;
    p TEXT;
BEGIN
    r := '';
    FOR p IN SELECT name
             FROM servers
             WHERE id =
                 ANY ((select allowed_sources FROM server_allowed_sources WHERE server_id = xid)::INT[])
        LOOP
            r := r || p || ',';
        END LOOP;
    RETURN rtrim(r, ',');
END;
$delim$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION in_submission_period(server_id integer) RETURNS BOOLEAN AS
$delim$
DECLARE
    t boolean;
BEGIN
    SELECT to_char(current_timestamp, 'HH24')::int >= start_submission_period
               AND
           to_char(current_timestamp, 'HH24')::int <= end_submission_period
    INTO t
    FROM servers
    WHERE id = server_id;
    RETURN t;
END;
$delim$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION xml_pretty(xml text)
    RETURNS xml AS
$$
SELECT xslt_process($1,
                    '<xsl:stylesheet version=\1.0\ xmlns:xsl=\http://www.w3.org/1999/XSL/Transform\>
                    <xsl:strip-space elements=\*\ />
                    <xsl:output method=\xml\ indent=\yes\ />
                    <xsl:template match=\node() | @*\>
                    <xsl:copy>
                    <xsl:apply-templates select=\node() | @*\ />
                    </xsl:copy>
                    </xsl:template>
                    </xsl:stylesheet>')::xml
$$ LANGUAGE SQL IMMUTABLE
                STRICT;

CREATE OR REPLACE FUNCTION is_valid_json(p_json text)
    RETURNS BOOLEAN
AS
$$
BEGIN
    return (p_json::json is not null);
EXCEPTION
    WHEN OTHERS THEN
        return false;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION pp_json(j TEXT, sort_keys BOOLEAN = TRUE, indent TEXT = '    ')
    RETURNS TEXT AS
$delim$
  import simplejson as json
  if not j:
      return ''
  return json.dumps(json.loads(j), sort_keys=sort_keys, indent=indent)
$delim$ LANGUAGE plpython3u;

CREATE OR REPLACE FUNCTION body_pprint(body text)
    RETURNS TEXT AS
$$
BEGIN
    IF xml_is_well_formed_document(body) THEN
        return xml_pretty(body)::text;
    ELSIF is_valid_json(body) THEN
        return pp_json(body, 't', '    ');
    ELSE
        return body;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- used to check is jsonb objects are identical
CREATE OR REPLACE FUNCTION jsonb_diff_val(val1 JSONB, val2 JSONB)
    RETURNS JSONB AS
$$
DECLARE
    result JSONB;
    v      RECORD;
BEGIN
    result = val1;
    FOR v IN SELECT * FROM jsonb_each(val2)
        LOOP
            IF result @> jsonb_build_object(v.key, v.value)
            THEN
                result = result - v.key;
            ELSIF result ? v.key THEN
                CONTINUE;
            ELSE
                result = result || jsonb_build_object(v.key, 'null');
            END IF;
        END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION geometry_geojson(geom geometry, lvl INT) RETURNS JSON AS
$$
DECLARE
    ret JSON;
BEGIN
    SELECT CASE
               WHEN geom IS NOT NULL THEN st_asgeojson(geom)::JSON
               ELSE
                   CASE
                       WHEN lvl = 1 THEN
                           NULL::JSON
                       WHEN lvl = 2 THEN
                           NULL::JSON
                       WHEN lvl = 3 THEN
                           -- '{"type": "Polygon", "coordinates": []}'::JSON
                           NULL::JSON
                       WHEN lvl = 4 THEN
                           -- '{"type": "MultiPolygon", "coordinates": []}'::JSON
                           NULL::JSON
                       WHEN lvl = 5 THEN
                           -- '{"type": "Point", "coordinates": []}'::JSON
                           NULL::JSON
                       END
               END
    INTO ret;
    RETURN ret;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_parent(i BIGINT) RETURNS JSON AS
$$
DECLARE
    parent TEXT := '';
    pid    BIGINT;
BEGIN
    SELECT parentid INTO pid FROM organisationunit WHERE id = i;
    IF FOUND THEN
        SELECT ('{ "id": "' || uid || '" }')::JSON
        INTO parent
        FROM organisationunit
        WHERE id = pid;
    END IF;

    RETURN parent;
END
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_requests_cc_status(int_array integer[]) RETURNS jsonb AS
$$
DECLARE
    i           integer;
    result_json jsonb := '{}';
BEGIN
    IF array_length(int_array, 1) IS NOT NULL THEN
        FOR i IN array_lower(int_array, 1) .. array_upper(int_array, 1)
            LOOP
                -- Create a JSON object for the current integer
                result_json := jsonb_set(
                        result_json,
                        ARRAY [(int_array)[i]]::text[],
                        '{
                          "status": "",
                          "errors": "",
                          "retries": 0,
                          "statusCode": "",
                          "response": ""
                        }'::jsonb,
                        true
                               );
            END LOOP;
    END IF;

    RETURN result_json;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger function
CREATE OR REPLACE FUNCTION after_request_insert_trigger_function()
    RETURNS TRIGGER AS
$$
BEGIN
    -- Call the generate_json_objects function with the inserted array
    UPDATE requests
    SET cc_servers_status = create_requests_cc_status(NEW.cc_servers)
    WHERE id = NEW.id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
CREATE TRIGGER after_request_insert_trigger
    AFTER INSERT
    ON requests
    FOR EACH ROW
EXECUTE PROCEDURE after_request_insert_trigger_function();

CREATE OR REPLACE FUNCTION status_of_dependence(reqId BIGINT) RETURNS TEXT AS
$delim$
DECLARE
    dep_status TEXT := '';
    dependence BIGINT;
BEGIN
    SELECT depends_on INTO dependence FROM requests WHERE id = reqId;
    IF dependence IS NOT NULL THEN
        SELECT status INTO dep_status FROM requests WHERE id = dependence;
        RETURN dep_status;
    END IF;
    RETURN dep_status;
END;
$delim$ LANGUAGE 'plpgsql';

CREATE OR REPLACE FUNCTION failed_cc_servers(servers integer[], servers_status jsonb) RETURNS integer[] AS
$delim$
DECLARE
    i              integer;
    failed_servers integer[] := '{}'::int[];
    status_code    text;
    status         text;
BEGIN
    IF array_length(servers, 1) IS NOT NULL THEN
        FOR i IN array_lower(servers, 1) .. array_upper(servers, 1)
            LOOP
                status_code := servers_status -> ((servers)[i])::text ->> 'statusCode';
                status := servers_status -> ((servers)[i])::text ->> 'status';
                IF status_code LIKE '4%' OR status_code LIKE '5%' OR status = '' THEN
                    failed_servers := array_append(failed_servers, servers[i]);
                END IF;

            END LOOP;
    END IF;

    RETURN failed_servers;
END;
$delim$ LANGUAGE plpgsql;

-- reset request's cc_servers_retries for cc_servers - targeting failed ones
CREATE OR REPLACE FUNCTION reset_request_cc_server_retries(reqId bigint, servers integer[]) RETURNS void AS
$delim$
DECLARE
    servers_status jsonb;
BEGIN
    SELECT cc_servers_status INTO servers_status FROM requests WHERE id = reqId;
    IF FOUND THEN
        IF array_length(servers, 1) IS NOT NULL THEN
            FOR i IN array_lower(servers, 1) .. array_upper(servers, 1)
                LOOP
                    servers_status :=
                            jsonb_set(servers_status, ARRAY [servers[i],'retries']::TEXT[], '0'::jsonb, false);
                    -- servers_status := jsonb_set(servers_status, ARRAY[servers[i],'statusCode']::TEXT[],'""'::jsonb, false);
                    -- servers_status := jsonb_set(servers_status, ARRAY[servers[i],'status']::TEXT[],'""'::jsonb, false);
                    EXECUTE 'UPDATE requests SET cc_servers_status = $1  WHERE id = $2' USING servers_status, reqId;
                END LOOP;
        END IF;
    END IF;


END;
$delim$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION is_point_in_polygon(lon FLOAT8, lat FLOAT8, polygon_id BIGINT)
    RETURNS BOOLEAN AS $$
BEGIN
    RETURN (
        -- SELECT ST_Contains(geometry, ST_SetSRID(ST_MakePoint(lon, lat), 4326))
        SELECT ST_Covers(geometry, ST_SetSRID(ST_MakePoint(lon, lat), 4326))
        FROM organisationunit
        WHERE id = polygon_id
        LIMIT 1
    );
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION generate_uid()
    RETURNS text

AS $function$
declare
    chars  text [] := '{0,1,2,3,4,5,6,7,8,9,a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p,q,r,s,t,u,v,w,x,y,z,A,B,C,D,E,F,G,H,I,J,K,L,M,N,O,P,Q,R,S,T,U,V,W,X,Y,Z}';
    result text := chars [11 + random() * (array_length(chars, 1) - 11)];
begin
    for i in 1..10 loop
            result := result || chars [1 + random() * (array_length(chars, 1) - 1)];
        end loop;
    return result;
end;
$function$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS dhis2_mappings
(
    id                    SERIAL PRIMARY KEY NOT NULL,
    uid                   TEXT               NOT NULL DEFAULT generate_uid(),
    name                  TEXT               NOT NULL,
    code                  TEXT               NOT NULL DEFAULT '',
    description           TEXT               NOT NULL DEFAULT '',
    dataset               TEXT               NOT NULL DEFAULT '',
    dhis2_name            TEXT               NOT NULL DEFAULT '',
    dataelement           TEXT               NOT NULL,
    category_option_combo TEXT,
    created               TIMESTAMPTZ                 DEFAULT CURRENT_TIMESTAMP,
    updated               TIMESTAMPTZ                 DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_dhis2_mappings_name ON dhis2_mappings (name);
CREATE INDEX IF NOT EXISTS idx_dhis2_mappings_uid ON dhis2_mappings (uid);
CREATE INDEX IF NOT EXISTS idx_dhis2_mappings_code ON dhis2_mappings (code);
CREATE INDEX IF NOT EXISTS idx_dhis2_mappings_dataset ON dhis2_mappings (dataset);
CREATE INDEX IF NOT EXISTS idx_dhis2_mappings_dhis2_name ON dhis2_mappings (dhis2_name);
CREATE INDEX IF NOT EXISTS idx_dhis2_mappings_dataelement ON dhis2_mappings (dataelement);

CREATE TABLE IF NOT EXISTS submission_log
(
    id              SERIAL PRIMARY KEY,
    submitted_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    payload         JSONB     NOT NULL,
    status          VARCHAR(20)        DEFAULT 'queued',
    retry_count     INTEGER   NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMP,
    task_id         VARCHAR(64),
    response        TEXT
);


CREATE INDEX IF NOT EXISTS idx_submission_log_status ON submission_log (status);
CREATE INDEX IF NOT EXISTS idx_submission_log_task_id ON submission_log (task_id);
CREATE INDEX IF NOT EXISTS idx_submission_log_submitted_at ON submission_log (submitted_at);
CREATE INDEX IF NOT EXISTS idx_submission_log_retry_count ON submission_log (retry_count);
CREATE INDEX IF NOT EXISTS idx_submission_log_last_attempt_at ON submission_log (last_attempt_at);



-- Data Follows
INSERT INTO servers (uid, name, username, password, ipaddress, url, auth_method, auth_token)
VALUES (generate_uid(), 'localhost', 'tester', 'foobar', '127.0.0.1', 'http://localhost:8080/test', 'Basic Auth', ''),
       (generate_uid(), 'dhis2', 'admin', 'district', '127.0.0.1', 'http://localhost/api/dataValueSets', 'Token',
        'd2pat_yrpULZwF9iltNDB3SxCTqUxTchRK5Byx0832006526');

INSERT INTO user_roles(name, description)
VALUES ('Administrator', 'For the Administrators'),
       ('SMS User', 'For SMS third party apps');

INSERT INTO user_role_permissions(user_role, sys_module, sys_perms)
VALUES ((SELECT id FROM user_roles WHERE name = 'Administrator'), 'Users', 'rmad');

INSERT INTO users(firstname, lastname, username, password, email, user_role, is_admin_user)
VALUES ('Samuel', 'Sekiwere', 'admin', crypt('@dm1n', gen_salt('bf')), 'sekiskylink@gmail.com',
        (SELECT id FROM user_roles WHERE name = 'Administrator'), 't');

INSERT INTO server_allowed_sources (server_id, allowed_sources)
VALUES ((SELECT id FROM servers where name = 'dhis2'),
        (SELECT array_agg(id) FROM servers WHERE name in ('localhost')));





-- ------------------------------------------------------------------------ --
-- Cron Expression Parser for PostgreSQL                                    --
--                                                                          --
-- (C) 2021-2022 Chris Mair <chris@1006.org>                                --
--                                                                          --
-- see LICENSE for license information                                      --
-- ------------------------------------------------------------------------ --


create schema if not exists cronexp;
drop function if exists cronexp.match(timestamp with time zone, text);
drop function if exists cronexp.expand_field(text, int, int);
drop function if exists cronexp.is_wellformed(text);


create or replace function cronexp.expand_field(field text, min int, max int)
    returns int[] as
$$
declare
    part   text;
    groups text[];
    m      int;
    n      int;
    k      int;
    ret    int[];
    tmp    int[];
begin

    -- step 1: basic parameter check

    if coalesce(field, '') = '' then
        raise exception 'invalid parameter "field"';
    end if;

    if min is null or max is null or min < 0 or max < 0 or min > max then
        raise exception 'invalid parameter(s) "min" or "max"';
    end if;

    -- step 2: handle special cases * and */k

    if field = '*' then
        select array_agg(x::int) into ret from generate_series(min, max) as x;
        return ret;
    end if;

    if field ~ '^\*/\d+$' then
        groups = regexp_matches(field, '^\*/(\d+)$');
        k := groups[1];
        if k < 1 or k > max then
            raise exception 'invalid range step: expected a step between 1 and %, got %', max, k;
        end if;
        select array_agg(x::int) into ret from generate_series(min, max, k) as x;
        return ret;
    end if;

    -- step 3: handle generic expression with values, lists or ranges

    ret := '{}'::int[];
    for part in select * from regexp_split_to_table(field, ',')
        loop
            if part ~ '^\d+$' then
                n := part;
                if n < min or n > max then
                    raise exception 'value out of range: expected values between % and %, got %', min, max, n;
                end if;
                ret = ret || n;
            elseif part ~ '^\d+-\d+$' then
                groups = regexp_matches(part, '^(\d+)-(\d+)$');
                m := groups[1];
                n := groups[2];
                if m > n then
                    raise exception 'inverted range bounds';
                end if;
                if m < min or m > max or n < min or n > max then
                    raise exception 'invalid range bound(s): expected bounds between % and %, got % and %', min, max, m, n;
                end if;
                select array_agg(x) into tmp from generate_series(m, n) as x;
                ret := ret || tmp;
            elseif part ~ '^\d+-\d+/\d+$' then
                groups = regexp_matches(part, '^(\d+)-(\d+)/(\d+)$');
                m := groups[1];
                n := groups[2];
                k := groups[3];
                if m > n then
                    raise exception 'inverted range bounds';
                end if;
                if m < min or m > max or n < min or n > max then
                    raise exception 'invalid range bound(s): expected bounds between % and %, got % and %', min, max, m, n;
                end if;
                if k < 1 or k > max then
                    raise exception 'invalid range step: expected a step between 1 and %, got %', max, k;
                end if;
                select array_agg(x) into tmp from generate_series(m, n, k) as x;
                ret := ret || tmp;
            else
                raise exception 'invalid expression';
            end if;
        end loop;

    select array_agg(x)
    into ret
    from (
             select distinct unnest(ret) as x
             order by x
         ) as sub;
    return ret;
end;
$$ language 'plpgsql' immutable;

create or replace function cronexp.match(ts timestamp with time zone, exp text)
    returns boolean as
$$
declare
    field_min int[] := '{ 0,  0,  1,  1, 0}';
    field_max int[] := '{59, 23, 31, 12, 7}';
    groups    text[];
    fields    int[];
    ts_parts  int[];

begin

    if ts is null then
        -- raise exception 'invalid parameter "ts": must not be null';
        return false;
    end if;

    if exp is null then
        -- raise exception 'invalid parameter "exp": must not be null';
        return false;
    end if;

    groups = regexp_split_to_array(trim(exp), '\s+');
    if array_length(groups, 1) != 5 then
        -- raise exception 'invalid parameter "exp": five space-separated fields expected';
        return false;
    end if;

    ts_parts[1] := date_part('minute', ts);
    ts_parts[2] := date_part('hour', ts);
    ts_parts[3] := date_part('day', ts);
    ts_parts[4] := date_part('month', ts);
    ts_parts[5] := date_part('dow', ts); -- Sunday = 0

    for n in 1..5
        loop
            fields := cronexp.expand_field(groups[n], field_min[n], field_max[n]);
            -- hack for DOW: fields might contain 0 or 7 for Sunday; if there's a 7, make sure there's a 0 too
            if n = 5 and array [7] <@ fields then
                fields := array [0] || fields;
            end if;
            if not array [ts_parts[n]] <@ fields then
                return false;
            end if;
        end loop;

    return true;
end
$$ language 'plpgsql' immutable;

