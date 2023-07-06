// deno-lint-ignore-file no-case-declarations
/** 
 *  - The code response should be only for bash or sh(later) and it should always 
 *    be returned as a markdown(?)
 * 
 *  - The system has access to system info like, current date, os, and other relavant info 
 *    that can be used to generate the code
 * 
 * **/



const code_generator_system_role = `
You are a helpful command-line interpreter. You receive natural language queries and you return the correspondent bash command. And only the command.
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


Deno.addSignalListener("SIGINT", () => {
    console.log("bye!");
    Deno.exit();
});


// get the query input from the user
const user_query = prompt('> ');

if(!user_query) {
    console.log('You need to ask a question');
    Deno.exit(1);  
}

console.log('processing...');

// make craft request to chat gpt api and sent it
const request = await fetch('https://api.openai.com/v1/chat/completions', {
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


const response = await request.json();

// console.log(response);


// mock for testing 
// const response = {
//     id: "chatcmpl-7ZHjtf5sWGuzLbzADZaykLS22jBPC",
//     object: "chat.completion",
//     created: 1688644025,
//     model: "gpt-3.5-turbo-0613",
//     choices: [
//         {
//             index: 0,
//             message: {
//                 role: "assistant",
//                 // content: "ls -la; say 'There you go little piggy!'"
//                 // content: `ffmpeg -i test.jpg -vf "scale=400: 400" output.jpg`
//                 content: "ffmpeg"
//             },
//             finish_reason: "stop"
//         }
//     ],
//     usage: { prompt_tokens: 112, completion_tokens: 14, total_tokens: 126 }
// }


//  print the response code to the screen
let teleprompt = `
    =========================================================================
                    Result
    =========================================================================
    |
    |
    |   ${response.choices[0].message.content}   
    |
    |
    -------------------------------------------------------------------------
    
        1: Run ↲   2: Review ♺   3: Explain 🎓   4: Copy 📋   5: Exit ⓧ

    -------------------------------------------------------------------------
`

let choice_menu = null;

while(isNaN(choice_menu) || choice_menu < 1 || choice_menu > 5){
    console.log(teleprompt);
    choice_menu = prompt('Choose an option[1-5]#')
}

const code_result = response.choices[0].message.content;

switch(choice_menu){
    case '1':
        console.log('running code...');
        
        const command = new Deno.Command('/bin/bash', {
            args: [
                "-c",
                code_result
            ],
        });       
        const { code, stdout, stderr } = await command.output();
        console.log(new TextDecoder().decode(stdout));

        break;
    case '2':
        console.log('revising code...');
        break;
    case '3':
        console.log('explaining code...');
        break;
    case '4':
        console.log('copying code...');

        break;
    default:
        console.log('bye!');
        Deno.exit();
}

