create database if not exists dtm_barrier /*!40100 DEFAULT CHARACTER SET utf8mb4 */;

drop table if exists dtm_barrier.barrier;
create table if not exists dtm_barrier.barrier(
  id int(11) PRIMARY KEY AUTO_INCREMENT,
  trans_type varchar(45) default '' ,
  gid varchar(128) default'',
  branch_id varchar(128) default '',
  branch_type varchar(45) default '',
  reason varchar(45) default '' comment 'the branch type who insert this record',
  result varchar(2047) default null comment 'the business result of this branch',
  create_time datetime DEFAULT now(),
  update_time datetime DEFAULT now(),
  key(create_time),
  key(update_time),
  UNIQUE key(gid, branch_id, branch_type)
);
