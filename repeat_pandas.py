#!/usr/bin/env python3

import pandas as pd
import sqlalchemy
import tempfile
import tarfile
import glob

DEFAULT_COLLECTIONS_FILE = "collections.db"


def load_collection(filepath, collection):
    with tempfile.TemporaryDirectory() as tmp_dir:
        with tarfile.open(filepath, 'r:gz') as tar:
            tar.extractall(tmp_dir)

        db_file = glob.glob("{0}/repeat-*/{1}".format(tmp_dir, DEFAULT_COLLECTIONS_FILE))[0]
        engine = sqlalchemy.create_engine("sqlite:///{0}".format(db_file),
                                          execution_options={"sqlite_raw_colnames": True})
        dfs = pd.read_sql_table(collection, engine)
        return dfs