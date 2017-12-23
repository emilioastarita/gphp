<?php

require_once __DIR__ . "/tolerant-php-parser/src/bootstrap.php";

define('T_NEW_LINE', -1);

$parser = new \Microsoft\PhpParser\Parser();

function usage($argv) {
    echo 'Usage: ' . $argv[0] . ' scan|parse|tokens file|directory'  . PHP_EOL;
    echo ' - scan: Scans file and prints tokens using tolerant-php-parser'. PHP_EOL;
    echo ' - parse: Parse file and prints AST using tolerant-php-parser'. PHP_EOL;
    echo ' - tokens: Scan file and prints tokens using token_get_all_nl'. PHP_EOL;
    var_dump($argv);
}

if (count($argv) !== 3) {
    usage($argv);
    exit(1);
}

if (in_array($argv[1], ['scan', 'parse', 'tokens'], true) === false) {
    usage($argv);
    exit(1);
}

if (!file_exists($argv[2])) {
    echo 'Err file doest not exists: ' . $argv[2] . PHP_EOL;
    exit(1);
}

if ($argv[1] === 'tokens') {
    $file = file_get_contents($argv[2]);
    echo "file:\n $file\n\n*******************\n";
    $tokens = token_get_all_nl($file);

    foreach ($tokens as $token)
    {
        if (is_array($token))
        {
            echo (token_name_nl($token[0]) . ': `' . $token[1] . '`'. PHP_EOL);
        }
        else
        {
            echo ('`' . $token . '`'. PHP_EOL);
        }
    }
} else if ($argv[1] === 'parse') {
    if (is_dir($argv[2])) {
        $rDir = new RecursiveDirectoryIterator($argv[2]);
        $iterator = new RecursiveIteratorIterator($rDir);
        $regex = new RegexIterator($iterator, '/^.+\.php$/i', RecursiveRegexIterator::GET_MATCH);
        foreach($regex as $filename) {
            echo $filename[0] .PHP_EOL;
            parseFile($filename[0]);
        }
    } else {
        parseFile($argv[2]);

    }
} else if ($argv[1] === 'scan') {
    scanFile($argv[2]);
}

function parseFile($file) {
    global $parser;
    $content = file_get_contents($file);
    $sourceFileNode = $parser->parseSourceFile($content);
    $tokens = str_replace("\r\n", "\n", json_encode($sourceFileNode, JSON_PRETTY_PRINT));
    echo $tokens;
}


function scanFile($file) {
    $content = file_get_contents($file);
    $GLOBALS["SHORT_TOKEN_SERIALIZE"] = false;
    $lexer = \Microsoft\PhpParser\TokenStreamProviderFactory::GetTokenStreamProvider($content);
    echo json_encode($lexer->getTokensArray());
}

function token_get_all_nl($source)
{
    $new_tokens = array();

    // Get the tokens
    $tokens = token_get_all($source);

    // Split newlines into their own tokens
    foreach ($tokens as $token)
    {
        $token_name = is_array($token) ? $token[0] : null;
        $token_data = is_array($token) ? $token[1] : $token;

        // Do not split encapsed strings or multiline comments
        if ($token_name == T_CONSTANT_ENCAPSED_STRING || substr($token_data, 0, 2) == '/*')
        {
            $new_tokens[] = array($token_name, $token_data);
            continue;
        }

        // Split the data up by newlines
        $split_data = preg_split('#(\r\n|\n)#', $token_data, -1, PREG_SPLIT_DELIM_CAPTURE | PREG_SPLIT_NO_EMPTY);

        foreach ($split_data as $data)
        {
            if ($data == "\r\n" || $data == "\n")
            {
                // This is a new line token
                $new_tokens[] = array(T_NEW_LINE, $data);
            }
            else
            {
                // Add the token under the original token name
                $new_tokens[] = is_array($token) ? array($token_name, $data) : $data;
            }
        }
    }

    return $new_tokens;
}

function token_name_nl($token)
{
    if ($token === T_NEW_LINE)
    {
        return 'T_NEW_LINE';
    }

    return token_name($token);
}


