%{
//
// Copyright (c) 2014 10X Genomics, Inc. All rights reserved.
//
// MRO grammar.
//

package syntax

import (
    "strconv"
    "strings"
)

%}

%union{
    global    *Ast
    srcfile   *SourceFile
    arr       int
    loc       int
    val       string
    modifiers *Modifiers
    dec       Dec
    decs      []Dec
    inparam   *InParam
    outparam  *OutParam
    retains   []*RetainParam
    stretains *RetainParams
    params    *Params
    res       *Resources
    par_tuple paramsTuple
    src       *SrcParam
    exp       Exp
    exps      []Exp
    rexp      *RefExp
    vexp      *ValExp
    kvpairs   map[string]Exp
    call      *CallStm
    calls     []*CallStm
    binding   *BindStm
    bindings  *BindStms
    retstm    *ReturnStm
    plretains *PipelineRetains
    reflist   []*RefExp
    includes  []*Include
}

%type <includes>  includes
%type <val>       id id_list type help type src_lang type outname
%type <modifiers> modifiers
%type <arr>       arr_list
%type <dec>       dec stage
%type <decs>      dec_list
%type <inparam>   in_param
%type <outparam>  out_param
%type <retains>   stage_retain_list
%type <stretains> stage_retain
%type <reflist>   pipeline_retain_list
%type <plretains> pipeline_retain
%type <params>    in_param_list out_param_list
%type <par_tuple> split_param_list
%type <src>       src_stm
%type <exp>       exp
%type <rexp>      ref_exp
%type <vexp>      val_exp bool_exp
%type <exps>      exp_list
%type <kvpairs>   kvpair_list
%type <call>      call_stm
%type <calls>     call_stm_list
%type <binding>   bind_stm modifier_stm
%type <bindings>  bind_stm_list modifier_stm_list
%type <retstm>    return_stm
%type <res>       resources resource_list

%token SKIP COMMENT INVALID
%token SEMICOLON COLON COMMA EQUALS
%token LBRACKET RBRACKET LPAREN RPAREN LBRACE RBRACE
%token SWEEP RETURN SELF
%token <val> FILETYPE STAGE PIPELINE CALL SPLIT USING RETAIN
%token <val> LOCAL PREFLIGHT VOLATILE DISABLED STRICT
%token IN OUT SRC AS
%token <val> THREADS MEM_GB SPECIAL
%token <val> ID LITSTRING NUM_FLOAT NUM_INT DOT
%token <val> PY GO SH EXEC COMPILED
%token <val> MAP INT STRING FLOAT PATH BOOL TRUE FALSE NULL DEFAULT
%token INCLUDE_DIRECTIVE

%%
file
    : includes dec_list
        {{
            global := NewAst($2, nil, $<srcfile>2)
            global.Includes = $1
            mmlex.(*mmLexInfo).global = global
        }}
    | includes dec_list call_stm
        {{
            global := NewAst($2, $3, $<srcfile>2)
            global.Includes = $1
            mmlex.(*mmLexInfo).global = global
        }}
    | includes call_stm
        {{
            global := NewAst([]Dec{}, $2, $<srcfile>2)
            global.Includes = $1
            mmlex.(*mmLexInfo).global = global
        }}
    | dec_list
        {{
            global := NewAst($1, nil, $<srcfile>1)
            mmlex.(*mmLexInfo).global = global
        }}
    | dec_list call_stm
        {{
            global := NewAst($1, $2, $<srcfile>1)
            mmlex.(*mmLexInfo).global = global
        }}
    | call_stm
        {{
            global := NewAst([]Dec{}, $1, $<srcfile>1)
            mmlex.(*mmLexInfo).global = global
        }}
    ;

includes
    : includes INCLUDE_DIRECTIVE LITSTRING
        {{ $$ = append($1, &Include{NewAstNode($<loc>2, $<srcfile>2), unquote($3)}) }}
    | INCLUDE_DIRECTIVE LITSTRING
        {{ $$ = []*Include{
              &Include{
                  Node: NewAstNode($<loc>1, $<srcfile>1),
                  Value: unquote($2),
              },
           }
        }}

dec_list
    : dec_list dec
        {{ $$ = append($1, $2) }}
    | dec
        {{ $$ = []Dec{$1} }}
    ;

dec
    : FILETYPE id_list SEMICOLON
        {{ $$ = &UserType{NewAstNode($<loc>2, $<srcfile>2), $2} }}
    | stage
    | PIPELINE id LPAREN in_param_list out_param_list RPAREN LBRACE call_stm_list return_stm pipeline_retain RBRACE
        {{ $$ = &Pipeline{NewAstNode($<loc>2, $<srcfile>2), $2, $4, $5, $8, &Callables{[]Callable{}, map[string]Callable{}}, $9, $10} }}
    ;

stage
    : STAGE id LPAREN in_param_list out_param_list src_stm RPAREN split_param_list resources stage_retain
        {{ $$ = &Stage{
                Node: NewAstNode($<loc>2, $<srcfile>2),
                Id:  $2,
                InParams: $4,
                OutParams: $5,
                Src: $6,
                ChunkIns: $8.Ins,
                ChunkOuts: $8.Outs,
                Split: $8.Present,
                Resources: $9,
                Retain: $10,
           } }}
   ;

resources
    :
        {{ $$ = nil }}
    | USING LPAREN resource_list RPAREN
        {{
             $3.Node = NewAstNode($<loc>1, $<srcfile>1)
             $$ = $3
         }}
    ;

resource_list
    :
        {{ $$ = &Resources{} }}
    | resource_list THREADS EQUALS NUM_INT COMMA
        {{
            n := NewAstNode($<loc>2, $<srcfile>2)
            $1.ThreadNode = &n
            i, _ := strconv.ParseInt($4, 0, 64)
            $1.Threads = int(i)
            $$ = $1
        }}
    | resource_list MEM_GB EQUALS NUM_INT COMMA
        {{
            n := NewAstNode($<loc>2, $<srcfile>2)
            $1.MemNode = &n
            i, _ := strconv.ParseInt($4, 0, 64)
            $1.MemGB = int(i)
            $$ = $1
        }}
    | resource_list SPECIAL EQUALS LITSTRING COMMA
        {{
            n := NewAstNode($<loc>2, $<srcfile>2)
            $1.SpecialNode = &n
            $1.Special = $4
            $$ = $1
        }}
    | resource_list VOLATILE EQUALS STRICT COMMA
        {{
            n := NewAstNode($<loc>2, $<srcfile>2)
            $1.VolatileNode = &n
            $1.StrictVolatile = true
            $$ = $1
        }}
    ;

stage_retain
    :
        {{ $$ = nil }}
    | RETAIN LPAREN stage_retain_list RPAREN
        {{
             $$ = &RetainParams{
                Node: NewAstNode($<loc>1, $<srcfile>1),
                Params: $3,
             }
         }}
    ;

stage_retain_list
    :
        {{ $$ = nil }}
    | stage_retain_list id COMMA
        {{
            $$ = append($1, &RetainParam{
                Node: NewAstNode($<loc>2, $<srcfile>2),
                Id: $2,
            })
        }}
    ;


id_list
    : id_list DOT id
        {{ $$ = $1 + $2 + $3 }}
    | id
    ;

arr_list
    :
        {{ $$ = 0 }}
    | arr_list LBRACKET RBRACKET
        {{ $$ += 1 }}
    ;

in_param_list
    :
        {{ $$ = &Params{[]Param{}, map[string]Param{}} }}
    | in_param_list in_param
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

in_param
    : IN type arr_list id help COMMA
        {{ $$ = &InParam{NewAstNode($<loc>1, $<srcfile>1), $2, $3, $4, unquote($5), false } }}
    | IN type arr_list id COMMA
        {{ $$ = &InParam{NewAstNode($<loc>1, $<srcfile>1), $2, $3, $4, "", false } }}
    ;

out_param_list
    :
        {{ $$ = &Params{[]Param{}, map[string]Param{}} }}
    | out_param_list out_param
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

out_param
    : OUT type arr_list COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<srcfile>1), $2, $3, "default", "", "", false } }}
    | OUT type arr_list help COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<srcfile>1), $2, $3, "default", unquote($4), "", false } }}
    | OUT type arr_list help outname COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<srcfile>1), $2, $3, "default", unquote($4), unquote($5), false } }}
    | OUT type arr_list id COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<srcfile>1), $2, $3, $4, "", "", false } }}
    | OUT type arr_list id help COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<srcfile>1), $2, $3, $4, unquote($5), "", false } }}
    | OUT type arr_list id help outname COMMA
        {{ $$ = &OutParam{NewAstNode($<loc>1, $<srcfile>1), $2, $3, $4, unquote($5), unquote($6), false } }}
    ;

src_stm
    : SRC src_lang LITSTRING COMMA
        {{ stagecodeParts := strings.Split(unquote($3), " ")
           $$ = &SrcParam{
               NewAstNode($<loc>1, $<srcfile>1),
               StageLanguage($2),
               stagecodeParts[0],
               stagecodeParts[1:],
           } }}
    ;

help
    : LITSTRING
    ;

outname
    : LITSTRING
    ;

type
    : INT
    | STRING
    | PATH
    | FLOAT
    | BOOL
    | MAP
    | id_list
    ;

src_lang
    : PY
    | EXEC
    | COMPILED
    //| GO
    //| SH
    ;

split_param_list
    :
        {{
            $$ = paramsTuple{
                false,
                &Params{[]Param{}, map[string]Param{}},
                &Params{[]Param{}, map[string]Param{}},
            }
        }}
    | SPLIT USING LPAREN in_param_list out_param_list RPAREN
        {{ $$ = paramsTuple{true, $4, $5} }}
    | SPLIT LPAREN in_param_list out_param_list RPAREN
        {{ $$ = paramsTuple{true, $3, $4} }}
    ;

return_stm
    : RETURN LPAREN bind_stm_list RPAREN
        {{ $$ = &ReturnStm{NewAstNode($<loc>1, $<srcfile>1), $3} }}
    ;

pipeline_retain
    :
        {{ $$ = nil }}
    | RETAIN LPAREN pipeline_retain_list RPAREN
        {{ $$ = &PipelineRetains{NewAstNode($<loc>1, $<srcfile>1), $3} }}

pipeline_retain_list
    :
        {{ $$ = nil }}
    | pipeline_retain_list ref_exp COMMA
        {{ $$ = append($1, $2) }}

call_stm_list
    : call_stm_list call_stm
        {{ $$ = append($1, $2) }}
    | call_stm
        {{ $$ = []*CallStm{$1} }}
    ;

call_stm
    : CALL modifiers id LPAREN bind_stm_list RPAREN
        {{ $$ = &CallStm{NewAstNode($<loc>1, $<srcfile>1), $2, $3, $3, $5} }}
    | CALL modifiers id AS id LPAREN bind_stm_list RPAREN
        {{ $$ = &CallStm{NewAstNode($<loc>1, $<srcfile>1), $2, $5, $3, $7} }}
    | call_stm USING LPAREN modifier_stm_list RPAREN
        {{
            $1.Modifiers.Bindings = $4
            $$ = $1
        }}
    ;

modifiers
    :
      {{ $$ = &Modifiers{} }}
    | modifiers LOCAL
      {{ $$.Local = true }}
    | modifiers PREFLIGHT
      {{ $$.Preflight = true }}
    | modifiers VOLATILE
      {{ $$.Volatile = true }}
    ;

modifier_stm_list
    :
        {{ $$ = &BindStms{NewAstNode($<loc>0, $<srcfile>0), []*BindStm{}, map[string]*BindStm{}} }}
    | modifier_stm_list modifier_stm
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

modifier_stm
    : LOCAL EQUALS bool_exp COMMA
        {{ $$ = &BindStm{NewAstNode($<loc>1, $<srcfile>1), $1, $3, false, ""} }}
    | PREFLIGHT EQUALS bool_exp COMMA
        {{ $$ = &BindStm{NewAstNode($<loc>1, $<srcfile>1), $1, $3, false, ""} }}
    | VOLATILE EQUALS bool_exp COMMA
        {{ $$ = &BindStm{NewAstNode($<loc>1, $<srcfile>1), $1, $3, false, ""} }}
    | DISABLED EQUALS ref_exp COMMA
        {{ $$ = &BindStm{NewAstNode($<loc>1, $<srcfile>1), $1, $3, false, ""} }}

bind_stm_list
    :
        {{ $$ = &BindStms{NewAstNode($<loc>0, $<srcfile>0), []*BindStm{}, map[string]*BindStm{}} }}
    | bind_stm_list bind_stm
        {{
            $1.List = append($1.List, $2)
            $$ = $1
        }}
    ;

bind_stm
    : id EQUALS exp COMMA
        {{ $$ = &BindStm{NewAstNode($<loc>1, $<srcfile>1), $1, $3, false, ""} }}
    | id EQUALS SWEEP LPAREN exp_list COMMA RPAREN COMMA
        {{ $$ = &BindStm{NewAstNode($<loc>1, $<srcfile>1), $1, &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindArray, Value: $5}, true, ""} }}
    | id EQUALS SWEEP LPAREN exp_list RPAREN COMMA
        {{ $$ = &BindStm{NewAstNode($<loc>1, $<srcfile>1), $1, &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindArray, Value: $5}, true, ""} }}
    ;

exp_list
    : exp_list COMMA exp
        {{ $$ = append($1, $3) }}
    | exp
        {{ $$ = []Exp{$1} }}
    ;

kvpair_list
    : kvpair_list COMMA LITSTRING COLON exp
        {{
            $1[unquote($3)] = $5
            $$ = $1
        }}
    | LITSTRING COLON exp
        {{ $$ = map[string]Exp{unquote($1): $3} }}
    ;

exp
    : val_exp
        {{ $$ = $1 }}
    | ref_exp
        {{ $$ = $1 }}

val_exp
    : LBRACKET exp_list RBRACKET
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindArray, Value: $2} }}
    | LBRACKET exp_list COMMA RBRACKET
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindArray, Value: $2} }}
    | LBRACKET RBRACKET
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindArray, Value: []Exp{}} }}
    | LBRACE RBRACE
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindMap, Value: map[string]interface{}{}} }}
    | LBRACE kvpair_list RBRACE
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindMap, Value: $2} }}
    | LBRACE kvpair_list COMMA RBRACE
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindMap, Value: $2} }}
    | NUM_FLOAT
        {{  // Lexer guarantees parseable float strings.
            f, _ := strconv.ParseFloat($1, 64)
            $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindFloat, Value: f }
        }}
    | NUM_INT
        {{  // Lexer guarantees parseable int strings.
            i, _ := strconv.ParseInt($1, 0, 64)
            $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindInt, Value: i }
        }}
    | LITSTRING
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindString, Value: unquote($1)} }}
    | bool_exp
    | NULL
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindNull, Value: nil} }}
    ;

bool_exp
    : TRUE
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindBool, Value: true} }}
    | FALSE
        {{ $$ = &ValExp{Node:NewAstNode($<loc>1, $<srcfile>1), Kind: KindBool, Value: false} }}

ref_exp
    : id DOT id
        {{ $$ = &RefExp{NewAstNode($<loc>1, $<srcfile>1), KindCall, $1, $3} }}
    | id
        {{ $$ = &RefExp{NewAstNode($<loc>1, $<srcfile>1), KindCall, $1, "default"} }}
    | SELF DOT id
        {{ $$ = &RefExp{NewAstNode($<loc>1, $<srcfile>1), KindSelf, $3, ""} }}
    ;

id
    : ID
    | COMPILED
    | DISABLED
    | EXEC
    | FILETYPE
    | LOCAL
    | MEM_GB
    | PREFLIGHT
    | RETAIN
    | SPECIAL
    | SPLIT
    | STRICT
    | THREADS
    | USING
    | VOLATILE
    ;
%%
