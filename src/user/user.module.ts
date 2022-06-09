import { Module } from '@nestjs/common';
import { DatastoreModule } from 'src/datastore/datastore.module';
import { UserService } from './user.service';

@Module({
  imports: [DatastoreModule],
  providers: [UserService],
  exports: [UserService]
})
export class UserModule {}
