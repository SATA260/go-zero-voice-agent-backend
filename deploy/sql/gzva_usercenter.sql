CREATE TABLE `user`
(
    `id`          bigint PRIMARY KEY,
    `create_time` timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `delete_time` timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state`   smallint     NOT NULL DEFAULT 0,
    `version`     bigint       NOT NULL DEFAULT 0 COMMENT '版本号',
    `email`       varchar(255) NOT NULL DEFAULT '',
    `password`    varchar(255) NOT NULL DEFAULT '',
    `nickname`    varchar(255) NOT NULL DEFAULT '',
    `sex`         smallint     NOT NULL DEFAULT 0 COMMENT '性别 0:男 1:女',
    `avatar`      varchar(255) NOT NULL DEFAULT '',
    `info`        varchar(255) NOT NULL DEFAULT '',
    CONSTRAINT `idx_mobile` UNIQUE (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户表';



CREATE TABLE `user_auth`
(
    `id`          bigint PRIMARY KEY,
    `create_time` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `delete_time` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `del_state`   smallint    NOT NULL DEFAULT 0,
    `version`     bigint      NOT NULL DEFAULT 0 COMMENT '版本号',
    `user_id`     bigint      NOT NULL DEFAULT 0,
    `auth_key`    varchar(64) NOT NULL DEFAULT '' COMMENT '平台唯一id',
    `auth_type`   varchar(12) NOT NULL DEFAULT '' COMMENT '平台类型',
    CONSTRAINT `idx_type_key` UNIQUE (`auth_type`, `auth_key`),
    CONSTRAINT `idx_userId_key` UNIQUE (`user_id`, `auth_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='用户授权表';
