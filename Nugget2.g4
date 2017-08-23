grammar Nugget2;

@header {
    // import "../NTypes"
}

prog: (define_assign | operation_on_singleton | NL )*
        EOF
;

define_assign:   define |
                 assign |
                 singleton_var
;

define: ID nugget_type LISTOP?;

assign: ID '=' STRING asType ('|' nugget_action)* |
        ID '=' ID ('|' nugget_action)*
;

operation_on_singleton: singleton_op '(' ID ')'
;

singleton_op: ('type' | 'print' | 'size');

singleton_var: ID;

asType: 'as' nugget_type;

nugget_type: 'string'  |
      'sha1'       |
      'md5'        |
      'ntfs'       |
      'file'       |
      'packet'     |
      'nettraffic' |
      'pcap'       |
      'exifinfo'
;

nugget_action: action_word (ID)?
;

action_word:
        filter    |
        'extract' |
        'sha1'    |
        'md5'
;

filter :
    'filter' filter_term (',' filter_term)*
;

filter_term: ID COMPOP STRING
;

COMPOP: ('>' | '<' | '>=' | '<=' | '==');
LISTOP: '[]';

INT : [0-9]+;
ID : [a-zA-Z]+;
STRING: '"' ('""'|~'"')* '"';

WS : [ \t\r\n]+ -> skip;
NL : '\r'? '\n';

LINE_COMMENT
    :   '//' ~[\r\n]* -> skip
    ;
