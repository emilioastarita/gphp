<?php

define('T_NEW_LINE', -1);

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

$file = file_get_contents($argv[1]);
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

?>