DROP TABLE IF EXISTS data_quality_results;
DROP TABLE IF EXISTS data_quality_rules;
DROP TABLE IF EXISTS pipeline_step_runs;
DROP TABLE IF EXISTS pipeline_runs;
DROP TABLE IF EXISTS triggers;

ALTER TABLE IF EXISTS pipelines
    DROP CONSTRAINT IF EXISTS pipelines_current_version_id_fkey;

DROP TABLE IF EXISTS pipeline_versions;
DROP TABLE IF EXISTS pipelines;
DROP TABLE IF EXISTS datasets;
DROP TABLE IF EXISTS connectors;
DROP TABLE IF EXISTS environments;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS department_members;
DROP TABLE IF EXISTS departments;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS orgs;
DROP TABLE IF EXISTS users;
