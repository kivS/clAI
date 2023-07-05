/** 
 *  - The code response should be only for bash or sh(later) and it should always 
 *    be returned as a markdown(?)
 * 
 *  - The system has access to system info like, current date, os, and other relavant info 
 *    that can be used to generate the code
 * 
 * **/



const code_generator_system_role = `
You are a helpful  command-line interpreter. You receive natural language queries and you return the correspondent bash command. And only the command.
You have access to some information about the system you are returning the command for.
===
OS: ${Deno.build.os}
ARCH: ${Deno.build.arch}
CURRENT_DATE: ${new Date()}
===

Example:
USER: how to list files?

ASSISTANT:
ls - la
`;

const OPENAI_API_KEY = Deno.env.get('OPENAI_API_KEY');



// get the query input from the user
let user_query = prompt('> ');

console.log('processing...');

// make craft request to chat gpt api and sent it
const response = await fetch('https://api.openai.com/v1/chat/completions', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${OPENAI_API_KEY}`
    },
    body: JSON.stringify({
        "model": "gpt-3.5-turbo",
        "messages": [
            { "role": "system", "content": code_generator_system_role },
            { "role": "user", "content": user_query }
        ]
    })

})

await response.json().then(data => {
    console.log(data);
})


//  print the response code to the screen

// the the user the option to either run the generated code, explain it, revise the query, copy the generated code to the clipboard, or exit the program