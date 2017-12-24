<?php

require_once __DIR__ . "/tolerant-php-parser/src/bootstrap.php";

define('T_NEW_LINE', -1);

$parser = new \Microsoft\PhpParser\Parser();

function usage($argv) {
    echo 'Usage: ' . $argv[0] . ' scan|parse|tokens file|directory'  . PHP_EOL;
    echo ' - scan: Scans file and prints tokens using tolerant-php-parser'. PHP_EOL;
    echo ' - parse: Parse file and prints AST using tolerant-php-parser'. PHP_EOL;
    echo ' - tokens: Scan file and prints tokens using token_get_all_nl'. PHP_EOL;
    echo ' - gencase-parser: Generates .tree with tolerant-php-parser'. PHP_EOL;
    echo ' - gencase-lexer: Generates .tokens with tolerant-php-parser'. PHP_EOL;
    var_dump($argv);
}

$validCommands = ['scan', 'parse', 'tokens', 'gencase-parser', 'gencase-tokens'];


if (count($argv) !== 3) {
    usage($argv);
    exit(1);
}

$command = $argv[1];
$filename = $argv[2];


if (in_array($command, $validCommands, true) === false) {
    usage($argv);
    exit(1);
}

if (!file_exists($filename)) {
    echo 'Err file doest not exists: ' . $filename . PHP_EOL;
    exit(1);
}

if ($command === 'tokens') {
    tokens($filename);
} else if ($command === 'parse') {
    if (is_dir($filename)) {
        parseDir($filename);
    } else {
        echo parseFile($filename);
    }
} else if ($command === 'gencase-tokens') {
    $tokens = scanFile($filename);
    $tokensFilename = $filename . ".tokens";
    if (file_put_contents($tokensFilename, $tokens) === false) {
        echo 'Err writting tokens file: ' . $tokensFilename . PHP_EOL;
        exit(1);
    }
} else if ($command === 'gencase-parser') {
    $tree = parseFile($filename);
    $treeFilename = $filename . ".tree";
    if (file_put_contents($treeFilename, $tree) === false) {
        echo 'Err writting tree file: ' . $treeFilename . PHP_EOL;
        exit(1);
    }
} else if ($command === 'scan') {
    echo scanFile($filename);
}

function tokens($filename) {
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

    $file = file_get_contents($filename);
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
}

function parseDir($dir) {
    $rDir = new RecursiveDirectoryIterator($dir);
    $iterator = new RecursiveIteratorIterator($rDir);
    $regex = new RegexIterator($iterator, '/^.+\.php$/i', RecursiveRegexIterator::GET_MATCH);
    foreach($regex as $filename) {
        echo $filename[0] .PHP_EOL;
        echo parseFile($filename[0]);
    }
}

function parseFile($file) {
    global $parser;
    $content = file_get_contents($file);
    $sourceFileNode = $parser->parseSourceFile($content);
    $tokens = str_replace("\r\n", "\n", json_encode($sourceFileNode, JSON_PRETTY_PRINT));
    return $tokens;
}


function scanFile($file) {
    $content = file_get_contents($file);
    $GLOBALS["SHORT_TOKEN_SERIALIZE"] = false;
    $lexer = \Microsoft\PhpParser\TokenStreamProviderFactory::GetTokenStreamProvider($content);
    return json_encode($lexer->getTokensArray());
}



