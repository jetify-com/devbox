from google.cloud import spanner
import argparse
import os

os.environ["SPANNER_EMULATOR_HOST"] = "localhost:1234"

OPERATION_TIMEOUT_SECONDS = 240


def create_instance(instance_id):
    """Creates an instance."""
    spanner_client = spanner.Client(project="test-project")
    config_name = "{}/instanceConfigs/regional-us-central1".format(
        spanner_client.project_name
    )

    instance = spanner_client.instance(
        instance_id,
        configuration_name=config_name,
        display_name="This is a display name.",
        node_count=1,
        labels={
            "cloud_spanner_samples": "true",
            "sample_name": "snippets-create_instance-explicit",
        },
    )

    operation = instance.create()
    operation.result(OPERATION_TIMEOUT_SECONDS)
    print("Created instance {}".format(instance_id))


def create_database(instance_id, database_id):
    """Creates a database and tables for sample data."""
    spanner_client = spanner.Client(project="test-project")
    instance = spanner_client.instance(
        instance_id,
    )

    database = instance.database(
        database_id,
        ddl_statements=[
            """CREATE TABLE Singers (
            SingerId     INT64 NOT NULL,
            FirstName    STRING(1024),
            LastName     STRING(1024),
            SingerInfo   BYTES(MAX),
            FullName   STRING(2048) AS (
                ARRAY_TO_STRING([FirstName, LastName], " ")
            ) STORED
        ) PRIMARY KEY (SingerId)""",
            """CREATE TABLE Albums (
            SingerId     INT64 NOT NULL,
            AlbumId      INT64 NOT NULL,
            AlbumTitle   STRING(MAX)
        ) PRIMARY KEY (SingerId, AlbumId),
        INTERLEAVE IN PARENT Singers ON DELETE CASCADE""",
        ],
    )

    operation = database.create()
    operation.result(OPERATION_TIMEOUT_SECONDS)
    print("Created database {} on instance {}".format(database_id, instance_id))

parser = argparse.ArgumentParser(description="Switch between two functions")

parser.add_argument("--instance", action="store_true", help="create instance")
parser.add_argument("--database", action="store_true", help="create database")

args = parser.parse_args()
if args.instance:
    create_instance("test-instance")
elif args.database:
    create_database("test-instance", "test-database")
else:
    print("Please specify --instance or --database")
