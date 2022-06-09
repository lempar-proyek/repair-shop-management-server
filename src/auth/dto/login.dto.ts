import { IsNotEmpty, IsString } from "class-validator"

export class LoginDto {
    @IsNotEmpty()
    method: string

    @IsNotEmpty()
    @IsString()
    credential: string
}