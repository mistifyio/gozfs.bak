#define _GNU_SOURCE
#include <assert.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

#include <libnvpair.h>

#define stringify(s) xstringify(s)
#define xstringify(s) #s

#define fnvlist_add_double(l, n, v) assert(nvlist_add_double(l, n, v) == 0)
#define fnvlist_add_hrtime(l, n, v) assert(nvlist_add_hrtime(l, n, v) == 0)

// TODO: add a dump for an empty nvlist

static void print(nvlist_t *list, char *name) {
	char *buf = NULL;
	size_t blen;
	int err;
	if ((err = nvlist_pack(list, &buf, &blen, NV_ENCODE_XDR, 0)) != 0) {
		printf("error:%d\n", err);
	}
	printf("\t{name: \"%s\", payload: []byte(\"", name);
	unsigned int i = 0;
	for (; i < blen - 1; i++) {
		printf("\\x%02x", buf[i] & 0xFF);
	}
	printf("\\x%02x\")},\n", buf[i]);
}

char *stra(char *s, int n) {
	size_t sl = strlen(s) + 2;
	char scomma[sl];
	memset(scomma, '\0', sl);
	strcat(scomma, s);
	strcat(scomma, ",");
	sl -= 1; // remove accounting for '\0'

	size_t size = sl * n + 1; // is really 1 byte extra but makes later stuff easier
	s = calloc(1, size);
	assert(s);

	for (int i = 0; i < n; i++) {
		strcat(s, scomma);
	}
	s[size - 2] = '\0'; //overwrite trailing ',' with '\0'
	return s;
}

char *stru(unsigned long long i) {
	char *s = NULL;
	int err = asprintf(&s, "%llu", i);
	if (err == -1) {
		printf("asprintf error: %d\n", err);
		assert(err != -1);
	}
	return s;
}

char *stri(long long i) {
	char *s = NULL;
	int err = asprintf(&s, "%lld", i);
	if (err == -1) {
		printf("asprintf error: %d\n", err);
		assert(err != -1);
	}
	return s;
}

char *strf(double d) {
	char *s = NULL;
	int err = asprintf(&s, "%16.17g", d);
	if (err == -1) {
		printf("asprintf error: %d\n", err);
		assert(err != -1);
	}
	return s;
}

#define do_signed(lower, UPPER) do { \
	l = fnvlist_alloc(); \
	fnvlist_add_##lower(l, stri(UPPER##_MIN), UPPER##_MIN); \
	fnvlist_add_##lower(l, stri(UPPER##_MIN+1), UPPER##_MIN+1); \
	fnvlist_add_##lower(l, stri(UPPER##_MIN/2), UPPER##_MIN/2); \
	fnvlist_add_##lower(l, "-1", -1); \
	fnvlist_add_##lower(l, "0", 0); \
	fnvlist_add_##lower(l, "1", 1); \
	fnvlist_add_##lower(l, stri(UPPER##_MAX/2), UPPER##_MAX/2); \
	fnvlist_add_##lower(l, stri(UPPER##_MAX-1), UPPER##_MAX-1); \
	fnvlist_add_##lower(l, stri(UPPER##_MAX), UPPER##_MAX); \
	print(l, stringify(lower) "s"); \
	fnvlist_free(l); \
} while(0)

#define do_unsigned(lower, UPPER) do { \
	l = fnvlist_alloc(); \
	fnvlist_add_##lower(l, "0", 0); \
	fnvlist_add_##lower(l, "1", 1); \
	fnvlist_add_##lower(l, stru(UPPER##_MAX/2), (UPPER##_MAX/2)); \
	fnvlist_add_##lower(l, stru(UPPER##_MAX-1), (UPPER##_MAX-1)); \
	fnvlist_add_##lower(l, stru(UPPER##_MAX), (UPPER##_MAX)); \
	print(l, stringify(lower) "s"); \
	fnvlist_free(l); \
} while(0)

#define arrset(array, alen, val) do { \
	size_t i = 0; \
	for (i = 0; i < alen; i++) { \
		array[i] = val; \
	} \
} while (0)

#define do_signed_array(lower, UPPER) do { \
	l = fnvlist_alloc(); \
	size_t len = 5; \
	lower##_t array[len]; \
	arrset(array, len, UPPER##_MIN); fnvlist_add_##lower##_array(l, stra(stri(UPPER##_MIN), len), array, len); \
	arrset(array, len, UPPER##_MIN+1); fnvlist_add_##lower##_array(l, stra(stri(UPPER##_MIN+1), len), array, len); \
	arrset(array, len, UPPER##_MIN/2); fnvlist_add_##lower##_array(l, stra(stri(UPPER##_MIN/2), len), array, len); \
	arrset(array, len, -1); fnvlist_add_##lower##_array(l, stra("-1", len), array, len); \
	arrset(array, len, 0); fnvlist_add_##lower##_array(l, stra("0", len), array, len); \
	arrset(array, len, 1); fnvlist_add_##lower##_array(l, stra("1", len), array, len); \
	arrset(array, len, UPPER##_MAX/2); fnvlist_add_##lower##_array(l, stra(stri(UPPER##_MAX/2), len), array,  len); \
	arrset(array, len, UPPER##_MAX-1); fnvlist_add_##lower##_array(l, stra(stri(UPPER##_MAX-1), len), array, len); \
	arrset(array, len, UPPER##_MAX); fnvlist_add_##lower##_array(l, stra(stri(UPPER##_MAX), len), array, len); \
	print(l, stringify(lower)" array"); \
	fnvlist_free(l); \
} while(0) \

#define do_unsigned_array(lower, UPPER) do { \
	l = fnvlist_alloc(); \
	size_t len = 5; \
	lower##_t array[len]; \
	arrset(array, len, 0); fnvlist_add_##lower##_array(l, stra("0", len), array, len); \
	arrset(array, len, 1); fnvlist_add_##lower##_array(l, stra("1", len), array, len); \
	arrset(array, len, UPPER##_MAX/2); fnvlist_add_##lower##_array(l, stra(stru(UPPER##_MAX/2), len), array, len); \
	arrset(array, len, UPPER##_MAX-1); fnvlist_add_##lower##_array(l, stra(stru(UPPER##_MAX-1), len), array, len); \
	arrset(array, len, UPPER##_MAX); fnvlist_add_##lower##_array(l, stra(stru(UPPER##_MAX), len), array, len); \
	print(l, stringify(lower)" array"); \
	fnvlist_free(l); \
} while(0) \

#if 0
	BYTE_ARRAY
	HRTIME
	BOOLEAN_ARRAY
	DOUBLE

#endif
int main() {
	printf("package nv\n"
	       "\n"
	       "/* !!! GENERATED FILE DO NOT EDIT !!! */\n"
	       "\n"
	       "var good = []struct {\n"
	       "\tname    string\n"
	       "\tpayload []byte\n"
	       "}{\n");

	nvlist_t *l = fnvlist_alloc();
	fnvlist_add_boolean_value(l, "false", B_FALSE);
	fnvlist_add_boolean_value(l, "true", B_TRUE);
	print(l, "bools");
	fnvlist_free(l);


	{
		l = fnvlist_alloc();
		size_t len = 5;
		boolean_t array[len];
		arrset(array, len, B_FALSE); fnvlist_add_boolean_array(l, "false,false,false,false,false", array, len);
		arrset(array, len, B_TRUE); fnvlist_add_boolean_array(l, "true,true,true,true,true", array, len);
		print(l, "bool array");
		fnvlist_free(l);
	}

	l = fnvlist_alloc();
	fnvlist_add_byte(l, "0", 0);
	fnvlist_add_byte(l, "1", 1);
	fnvlist_add_byte(l, "-128", -128);
	fnvlist_add_byte(l, "127", 127);
	print(l, "bytes");
	fnvlist_free(l);

	{
		l = fnvlist_alloc();
		size_t len = 5;
		unsigned char array[len];
		arrset(array, len, 0); fnvlist_add_byte_array(l, "0,0,0,0,0", array, len);
		arrset(array, len, 1); fnvlist_add_byte_array(l, "1,1,1,1,1", array, len);
		arrset(array, len, -128); fnvlist_add_byte_array(l, "-128,-128,-128,-128,-128", array, len);
		arrset(array, len, 127); fnvlist_add_byte_array(l, "127,127,127,127,127", array, len);
		print(l, "byte array");
		fnvlist_free(l);
	}


	do_signed(int8, INT8);
	do_signed(int16, INT16);
	do_signed(int32, INT32);
	do_signed(int64, INT64);
	do_signed_array(int8, INT8);
	do_signed_array(int16, INT16);
	do_signed_array(int32, INT32);
	do_signed_array(int64, INT64);

	do_unsigned(uint8, UINT8);
	do_unsigned(uint16, UINT16);
	do_unsigned(uint32, UINT32);
	do_unsigned(uint64, UINT64);
	do_unsigned_array(uint8, UINT8);
	do_unsigned_array(uint16, UINT16);
	do_unsigned_array(uint32, UINT32);
	do_unsigned_array(uint64, UINT64);

	l = fnvlist_alloc();
	fnvlist_add_string(l, "0", "0");
	fnvlist_add_string(l, "1", "1");
	fnvlist_add_string(l, "HiMom", "HiMom");
	fnvlist_add_string(l, "\xff\"; DROP TABLE USERS;", "\xff\"; DROP TABLE USERS;");
	print(l, "strings");
	fnvlist_free(l);


	{
		char *array[] = {
			"0",
			"1",
			"HiMom",
			"\xff\"; DROP TABLE USERS;",
		};
		l = fnvlist_alloc();
		fnvlist_add_string_array(l, "0,1,HiMom,\xff\"; DROP TABLE USERS;", array, 4);
		print(l, "string array");
		fnvlist_free(l);
	}

	{
		do_signed(hrtime, INT64);
	}


	l = fnvlist_alloc();
	nvlist_t *le = fnvlist_alloc();
	//fnvlist_add_boolean_value(l, "false", B_FALSE);
	//fnvlist_add_boolean_value(l, "true", B_TRUE);
	fnvlist_add_boolean_value(le, "false", B_FALSE);
	fnvlist_add_boolean_value(le, "true", B_TRUE);
	fnvlist_add_nvlist(l, "nvlist", le);
	print(l, "nvlist");
	fnvlist_free(l);


	l = fnvlist_alloc();
	nvlist_t *larr[] = {le, le};
	fnvlist_add_nvlist_array(l, "list,list", larr, 2);
	print(l, "nvlist array");
	fnvlist_free(le);
	fnvlist_free(l);

	{
		l = fnvlist_alloc();
		fnvlist_add_double(l, strf(DBL_MIN), DBL_MIN);
		fnvlist_add_double(l, strf(DBL_MIN+1), DBL_MIN+1);
		fnvlist_add_double(l, strf(DBL_MIN/2), DBL_MIN/2);
		fnvlist_add_double(l, "-1", -1);
		fnvlist_add_double(l, "0", 0);
		fnvlist_add_double(l, "1", 1);
		fnvlist_add_double(l, strf(DBL_MAX/2), DBL_MAX/2);
		fnvlist_add_double(l, strf(DBL_MAX-1), DBL_MAX-1);
		fnvlist_add_double(l, strf(DBL_MAX), DBL_MAX);
		print(l, stringify(double) "s");
		fnvlist_free(l);
	}

	printf("}\n");
	return 0;
}
