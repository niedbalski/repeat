#!/usr/bin/env python3

import pandas as pd
import sqlalchemy
import tempfile
import tarfile
import glob

DEFAULT_COLLECTIONS_FILE = "collections.db"


class Collections: pass


def load(filepath):
    with tempfile.TemporaryDirectory() as tmp_dir:
        with tarfile.open(filepath, 'r:gz') as tar:
            
            import os
            
            def is_within_directory(directory, target):
                
                abs_directory = os.path.abspath(directory)
                abs_target = os.path.abspath(target)
            
                prefix = os.path.commonprefix([abs_directory, abs_target])
                
                return prefix == abs_directory
            
            def safe_extract(tar, path=".", members=None, *, numeric_owner=False):
            
                for member in tar.getmembers():
                    member_path = os.path.join(path, member.name)
                    if not is_within_directory(path, member_path):
                        raise Exception("Attempted Path Traversal in Tar File")
            
                tar.extractall(path, members, numeric_owner=numeric_owner) 
                
            
            safe_extract(tar, tmp_dir)

        db_file = glob.glob("{0}/repeat-*/{1}".format(tmp_dir, DEFAULT_COLLECTIONS_FILE))[0]
        engine = sqlalchemy.create_engine("sqlite:///{0}".format(db_file),
                                          execution_options={"sqlite_raw_colnames": True})

        engine.connect()

        inspector = sqlalchemy.inspect(engine)
        collections = Collections()

        for table in inspector.get_table_names():
            setattr(collections, table, pd.read_sql_table(table, engine))
        return collections