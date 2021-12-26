CREATE TABLE aaa (
    cint int primary key,
    cvarchar varchar(30),
    ctext text,
    creal real,
    cblob blob
);
INSERT INTO "aaa" VALUES(0, 'var1', 'text1', 0, "blob1");
INSERT INTO "aaa" VALUES(1, 'var2', 'test2', 1, "blob2");
INSERT INTO "aaa" VALUES(128, 'var3', 'test3', 128, "blob3");
INSERT INTO "aaa" VALUES(-128, 'var3', 'test3', -128, "blob3");
INSERT INTO "aaa" VALUES(9223372036854775807, 'var4', 'test4', 9223372036854775807, "blob4");
INSERT INTO "aaa" VALUES(-9223372036854775808, 'var5', 'test5', -9223372036854775808, "blob5");

-- CREATE TABLE aaa (
--     cint int primary key
-- );
-- INSERT INTO "aaa" VALUES(123);
