grant all on archondb.* to 'archonadmin'@'localhost' identified by 'psoadminpassword';

insert into account_data (username, password, email, registration_date, is_gm, is_active, team_id) 
values ('test', '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08', 'test@email.com', now(), true, true, 1);

insert into account_data (username, password, email, registration_date, is_gm, is_active, team_id) 
values ('rabble', '5f121158d2145c5d68a82ba2d2a8052e1fffc7ed14ae601befc91e339e387860', 'rabble@email.com', now(), false, true, 2);