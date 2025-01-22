# JonathanScriptCompiler
A compiler for JonathanScript( JS) ;)
1. The Lexer translates JonathanScript into tokens.
2. The Parser parses the tokens and generates the AST.
3. The Compiler compiles the AST into bytecode (instructions).
4. The Virtual Machine executes the bytecode. The run function decodes the instructions, processes them, and pushes the results onto the stack.

Execution process：
<table>
  <tr>
    <td></td>
    <td>Lexer</td>
    <td></td>
    <td>Parser</td>
    <td></td>
    <td>Compiler</td>
    <td></td>
    <td>VM</td>
    <td></td>
  </tr>
  <tr>
    <td>source</td>
    <td>----></td>
    <td>token</td>
    <td>-----></td>
    <td>AST</td>
    <td>-------></td>
    <td>code (Instructions []byte)</td>
    <td>--></td>
    <td>result</td>
  </tr>
</table>
