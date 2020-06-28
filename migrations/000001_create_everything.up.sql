BEGIN;
CREATE TABLE IF NOT EXISTS tabs(
                                   id INT AUTO_INCREMENT PRIMARY KEY,
                                   name TEXT NOT NULL UNIQUE,
                                   slug TEXT NOT NULL UNIQUE
) ENGINE=INNODB;

CREATE TABLE IF NOT EXISTS nodes (
                                     id INT AUTO_INCREMENT PRIMARY KEY,
                                     tab_id INT,
                                     slug TEXT NOT NULL UNIQUE,
                                     CONSTRAINT fk_tab FOREIGN KEY (tab_id) REFERENCES tabs(id)
) ENGINE=INNODB;
CREATE TABLE IF NOT EXISTS threads (
                                       id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                       title TEXT NOT NULL,
                                       node_id INT,
                                       CONSTRAINT fk_node FOREIGN KEY (node_id) REFERENCES nodes(id)
) ENGINE=INNODB;
CREATE TABLE IF NOT EXISTS users (
                                     id INT AUTO_INCREMENT PRIMARY KEY,
                                     username TEXT UNIQUE NOT NULL,
                                     passhash TEXT NULL NOT NULL,
                                     email TEXT NULL,
                                     avatar_url TEXT NULL
) ENGINE=INNODB;
CREATE TABLE IF NOT EXISTS posts (
                                     id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                     created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                     updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                                     content TEXT NOT NULL,
                                     user_id INT,
                                     CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id),
                                     thread_id BIGINT,
                                     CONSTRAINT fk_thread FOREIGN KEY (thread_id) REFERENCES threads(id)

) ENGINE=INNODB;

COMMIT;
