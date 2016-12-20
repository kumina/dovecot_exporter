# Dovecot exporter

This repository provides a `dovecot_exporter` utility that can be used
to scrape statistics from Dovecot and export them as Prometheus metrics.
It extracts metrics through Dovecot's
[stats module](https://wiki2.dovecot.org/Statistics).

The metrics provided by this exporter look like this:

```
dovecot_up{scope="user"} 1
dovecot_user_last_update{user="foo@example.com"} 1.482243627730987e+09
dovecot_user_mail_cache_hits{user="foo@example.com"} 298
dovecot_user_mail_lookup_attr{user="foo@example.com"} 4
dovecot_user_mail_lookup_path{user="foo@example.com"} 87
dovecot_user_mail_read_bytes{user="foo@example.com"} 176544
dovecot_user_mail_read_count{user="foo@example.com"} 83
dovecot_user_maj_faults{user="foo@example.com"} 0
dovecot_user_min_faults{user="foo@example.com"} 156053
dovecot_user_num_cmds{user="foo@example.com"} 565
dovecot_user_num_logins{user="foo@example.com"} 80
dovecot_user_read_bytes{user="foo@example.com"} 2.63032577e+08
dovecot_user_read_count{user="foo@example.com"} 73992
dovecot_user_reset_timestamp{user="foo@example.com"} 1.482239247e+09
dovecot_user_sys_cpu{user="foo@example.com"} 2.236
dovecot_user_user_cpu{user="foo@example.com"} 11.944
dovecot_user_vol_cs{user="foo@example.com"} 2981
dovecot_user_write_bytes{user="foo@example.com"} 961822
dovecot_user_write_count{user="foo@example.com"} 4906
```
