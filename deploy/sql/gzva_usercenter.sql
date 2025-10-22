create table gzva_usercenter.user
(
    id          bigint auto_increment
        primary key,
    create_time timestamp    default CURRENT_TIMESTAMP not null,
    update_time timestamp    default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP,
    delete_time timestamp    default CURRENT_TIMESTAMP not null,
    del_state   smallint     default 0                 not null,
    version     bigint       default 0                 not null comment '版本号',
    email       varchar(255) default ''                not null,
    password    varchar(255) default ''                not null,
    nickname    varchar(255) default ''                not null,
    sex         smallint     default 0                 not null comment '性别 0:男 1:女',
    avatar      varchar(255) default ''                not null,
    info        varchar(255) default ''                not null,
    constraint idx_mobile
        unique (email)
)
    comment '用户表';

create table gzva_usercenter.user_auth
(
    id          bigint auto_increment
        primary key,
    create_time timestamp   default CURRENT_TIMESTAMP not null,
    update_time timestamp   default CURRENT_TIMESTAMP not null on update CURRENT_TIMESTAMP,
    delete_time timestamp   default CURRENT_TIMESTAMP not null,
    del_state   smallint    default 0                 not null,
    version     bigint      default 0                 not null comment '版本号',
    user_id     bigint      default 0                 not null,
    auth_key    varchar(64) default ''                not null comment '平台唯一id',
    auth_type   varchar(12) default ''                not null comment '平台类型',
    constraint idx_type_key
        unique (auth_type, auth_key),
    constraint idx_userId_key
        unique (user_id, auth_type)
)
    comment '用户授权表';

