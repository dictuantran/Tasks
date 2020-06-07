CREATE TABLE task (
    id integer primary key AUTO_INCREMENT,
    title varchar(100),
    content text,
    task_status_id integer references status(id),   
    created_date timestamp,
    due_date timestamp,
    last_modified_at timestamp,
    finish_date timestamp,
    priority integer,
    cat_id integer references category(id),
    user_id integer references user(id), 
    hide int);
    
CREATE TABLE status (
    id integer primary key AUTO_INCREMENT,
    status varchar(50) not null
);
CREATE TABLE files(
    name varchar(1000) not null,
    autoName varchar(255) not null,
    user_id integer references user(id),   
    created_date timestamp
);
CREATE TABLE category(
    id integer primary key AUTO_INCREMENT,
    name varchar(1000) not null,
    user_id integer references user(id)    
);
CREATE TABLE comments(
    id integer primary key AUTO_INCREMENT,
    content text,
    taskID integer references task(id),   
    created datetime,
    user_id integer references user(id)    
 );
CREATE TABLE user (
    id integer primary key AUTO_INCREMENT,
    username varchar(100),
    password varchar(1000),
    email varchar(100)
);

insert into status(status) values('COMPLETE');
insert into status(status) values('PENDING');
insert into status(status) values('DELETED');

